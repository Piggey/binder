package binder

import "github.com/go-playground/validator/v10"

type Binder struct {
	validate *validator.Validate
}

func New(binderOpts ...binderOpt) *Binder {
	b := Binder{
		validate: validator.New(validator.WithRequiredStructEnabled()),
	}

	for _, binderOpt := range binderOpts {
		binderOpt(&b)
	}

	return &b
}

type binderOpt func(b *Binder)

func WithCustomValidatorInstance(v *validator.Validate) binderOpt {
	return func(b *Binder) {
		b.validate = v
	}
}
