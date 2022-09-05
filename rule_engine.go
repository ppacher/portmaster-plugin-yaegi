package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"reflect"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/safing/portmaster/plugin/framework"
	"github.com/safing/portmaster/plugin/shared/decider"
	"github.com/safing/portmaster/plugin/shared/proto"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

var Symbols = map[string]map[string]reflect.Value{}

type (
	RuleEngine struct {
		l           sync.RWMutex
		interpreter *interp.Interpreter

		rules []namedDecider
	}

	namedDecider struct {
		decider.Decider

		Name string
	}
)

func NewRuleEngine() *RuleEngine {
	engine := &RuleEngine{}

	engine.Reset()

	return engine
}

func (engine *RuleEngine) Reset() {
	engine.l.Lock()
	defer engine.l.Unlock()

	inter := interp.New(interp.Options{})
	inter.Use(stdlib.Symbols)
	inter.Use(Symbols)

	engine.interpreter = inter
}

func (engine *RuleEngine) LoadPaths(paths ...string) error {
	var multierr = new(multierror.Error)

	for _, path := range paths {
		glob := filepath.Join(path, "*.go")
		matches, err := filepath.Glob(glob)

		if err != nil {
			multierr.Errors = append(multierr.Errors, err)

			continue
		}

		for _, file := range matches {
			_, err := engine.interpreter.EvalPath(file)
			if err != nil {
				multierr.Errors = append(multierr.Errors, fmt.Errorf("%s: %w", file, err))

				continue
			}

			deciderRaw, err := engine.interpreter.Eval("DecideOnConnection")
			if err != nil {
				return err
			}

			rules, ok := deciderRaw.Interface().(func(context.Context, *proto.Connection) (proto.Verdict, string, error))
			if !ok {
				return fmt.Errorf("wrong type returned when accessing rules,  %T", deciderRaw.Interface())
			}

			engine.rules = append(engine.rules, namedDecider{
				Decider: framework.DeciderFunc(rules),
				Name:    filepath.Base(file),
			})
		}
	}

	return multierr.ErrorOrNil()
}

func (engine *RuleEngine) DecideOnConnection(ctx context.Context, conn *proto.Connection) (proto.Verdict, string, error) {
	engine.l.RLock()
	defer engine.l.RUnlock()

	for _, decider := range engine.rules {
		log.Printf("evaluating connection %s against rule %s", conn.GetId(), decider.Name)

		verdict, reason, err := decider.DecideOnConnection(ctx, conn)

		switch verdict {
		case proto.Verdict_VERDICT_UNDECIDED,
			proto.Verdict_VERDICT_UNDETERMINABLE:

			continue
		default:
			return verdict, reason, err
		}
	}

	return proto.Verdict_VERDICT_UNDECIDED, "", nil
}

var _ decider.Decider = new(RuleEngine)
