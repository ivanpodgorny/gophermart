package main

import (
	"context"
	"database/sql"
	"errors"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	v10validator "github.com/go-playground/validator/v10"
	"github.com/ivanpodgorny/gophermart/internal/client"
	"github.com/ivanpodgorny/gophermart/internal/config"
	"github.com/ivanpodgorny/gophermart/internal/entity"
	"github.com/ivanpodgorny/gophermart/internal/handler"
	"github.com/ivanpodgorny/gophermart/internal/middleware"
	"github.com/ivanpodgorny/gophermart/internal/migrations"
	"github.com/ivanpodgorny/gophermart/internal/repository"
	"github.com/ivanpodgorny/gophermart/internal/security"
	"github.com/ivanpodgorny/gophermart/internal/service"
	"github.com/ivanpodgorny/gophermart/internal/validator"
	"github.com/ivanpodgorny/gophermart/internal/worker"
	_ "github.com/jackc/pgx/v5/stdlib"
	"log"
	"net/http"
	"sync"
)

func main() {
	if err := Execute(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func Execute() error {
	cfg, err := config.NewBuilder().LoadFlags().LoadEnv().Build()
	if err != nil {
		return err
	}

	db, err := sql.Open("pgx", cfg.DatabaseURI())
	if err != nil {
		return err
	}

	defer func(db *sql.DB) {
		err = db.Close()
	}(db)

	if err := migrations.Up(db); err != nil {
		return err
	}

	validationEngine := v10validator.New()
	if err := validationEngine.RegisterValidation("luhn", validator.Luhn); err != nil {
		return err
	}

	var (
		ctx, cancel = context.WithCancel(context.Background())
		r           = chi.NewRouter()
		v           = validator.New(validationEngine)
		a           = security.NewAuthenticator(security.NewHMACSigner(cfg.HMACKey()), repository.NewToken(db))
		wg          = &sync.WaitGroup{}
		scj         = make(chan entity.StatusCheckJob, 8)
		scr         = make(chan entity.StatusCheckResult, 8)
		or          = repository.NewOrder(db)
		ac          = client.NewAccrual(cfg.AccrualSystemAddress())
		scw         = worker.NewStatusChecker(ctx, or, ac, scj, scr, wg, 4)
		ouw         = worker.NewOrderUpdater(or, scr, wg, 4)
		ss          = service.NewSignup(
			repository.NewUser(db),
			security.NewArgonHasher(security.DefaultHashConfig()),
			a,
		)
		os = service.NewOrder(or, scj)
		ts = service.NewTransaction(repository.NewTransaction(db))
		sh = handler.NewSignup(ss, v)
		oh = handler.NewOrder(os, a, v)
		th = handler.NewTransaction(ts, a, v)
	)

	defer func() {
		cancel()
		wg.Wait()
		close(scj)
		close(scr)
	}()

	scw.Do(ctx)
	ouw.Do(ctx)

	r.Use(chimiddleware.Recoverer)

	r.Route("/api/user", func(r chi.Router) {
		r.Post("/register", sh.Register)
		r.Post("/login", sh.Login)

		r.Group(func(r chi.Router) {
			r.Use(middleware.Authenticate(a))

			r.Post("/orders", oh.Create)
			r.Get("/orders", oh.GetAll)
			r.Get("/balance", th.GetBalance)
			r.Post("/balance/withdraw", th.Withdraw)
			r.Get("/withdrawals", th.GetWithdrawals)
		})
	})

	err = http.ListenAndServe(cfg.ServerAddress(), r)

	return err
}
