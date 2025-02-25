package store

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gen"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var defaultHandler *Handler

type handlerKey struct{}

type Config struct {
	Driver string
	DSN    string
}

type Handler struct {
	*gorm.DB
}

func NewFromDB(db *gorm.DB) *Handler {
	return &Handler{
		DB: db,
	}
}

func NewContext(ctx context.Context, h *Handler) context.Context {
	return context.WithValue(ctx, handlerKey{}, h)
}

func WithContext(ctx context.Context) *Handler {
	return ctx.Value(handlerKey{}).(*Handler)
}

func MustInit(cfg Config) *Handler {
	h, err := Init(cfg)
	if err != nil {
		panic(err)
	}

	return h
}

func Init(cfg Config) (*Handler, error) {
	if defaultHandler != nil {
		return defaultHandler, nil
	}

	logger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      true,  // Don't include params in the SQL log
			Colorful:                  false, // Disable color
		},
	)

	var (
		err error
		db  *gorm.DB
	)
	switch cfg.Driver {
	case "postgres":
		db, err = gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{
			Logger: logger,
		})
	default:
		panic("unknown driver")
	}
	if err != nil {
		return nil, err
	}

	defaultHandler = &Handler{
		DB: db,
	}
	return defaultHandler, err
}

type generateModel struct {
	cfg gen.Config
	f   func(g *gen.Generator)
}

var generateModels []*generateModel

func RegistGenerate(cfg gen.Config, f func(g *gen.Generator)) {
	generateModels = append(generateModels, &generateModel{
		cfg: cfg,
		f:   f,
	})
}

func Generate() {
	for _, gm := range generateModels {
		if gm.cfg.Mode == 0 {
			gm.cfg.Mode = gen.WithoutContext | gen.WithDefaultQuery | gen.WithQueryInterface
		}
		g := gen.NewGenerator(gm.cfg)
		// g.UseDB(h.DB)
		gm.f(g)
		g.Execute()
	}
}

func Transaction(f func(tx *Handler) error) error {
	return defaultHandler.Transaction(func(db *gorm.DB) error {
		return f(&Handler{
			DB: db,
		})
	})
}

func IsNotFoundErr(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
