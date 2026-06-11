package di

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

type container struct {
	mu        sync.RWMutex
	providers map[reflect.Type]reflect.Value
	instances map[reflect.Type]reflect.Value
	pending   []func() error
}

func newContainer() *container {
	return &container{
		providers: make(map[reflect.Type]reflect.Value),
		instances: make(map[reflect.Type]reflect.Value),
		pending:   []func() error{},
	}
}

// Provide 注册构造函数，支持 func() T 或 func() (T, error)
func (c *container) Provide(constructor any) {
	ctorType := reflect.TypeOf(constructor)
	if ctorType.Kind() != reflect.Func {
		panic(fmt.Errorf("di: constructor must be a function, got %s", typeFullName(ctorType)))
	}
	numOut := ctorType.NumOut()
	if numOut != 1 && numOut != 2 {
		panic(fmt.Errorf("di: constructor must return 1 or 2 values, got %d", numOut))
	}
	if numOut == 2 {
		if !ctorType.Out(1).Implements(reflect.TypeFor[error]()) {
			panic(fmt.Errorf("di: second return value must be error, got %v", typeFullName(ctorType.Out(1))))
		}
	}
	returnType := ctorType.Out(0)

	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.providers[returnType]; ok {
		panic(fmt.Errorf("di: already registered for type %v", typeFullName(returnType)))
	}
	c.providers[returnType] = reflect.ValueOf(constructor)
}

// Invoke 注册一个延迟执行的函数，参数自动注入，函数可返回 error
func (c *container) Invoke(fn any) {
	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func {
		panic(fmt.Errorf("di: Invoke requires a function, got %s", typeFullName(fnType)))
	}
	wrapper := func() error {
		fnVal := reflect.ValueOf(fn)
		numIn := fnType.NumIn()
		args := make([]reflect.Value, numIn)
		for i := range numIn {
			argType := fnType.In(i)
			argVal, err := c.resolve(argType, map[reflect.Type]bool{})
			if err != nil {
				return fmt.Errorf("di: cannot resolve argument %d (%v): %w", i, typeFullName(argType), err)
			}
			args[i] = argVal
		}
		results := fnVal.Call(args)
		if len(results) > 0 {
			if last := results[len(results)-1]; last.Type().Implements(reflect.TypeFor[error]()) {
				if !last.IsNil() {
					return last.Interface().(error)
				}
			}
		}
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pending = append(c.pending, wrapper)
}

// resolve 递归解析类型，支持 (T, error) 构造函数
func (c *container) resolve(t reflect.Type, visiting map[reflect.Type]bool) (reflect.Value, error) {
	if visiting[t] {
		return reflect.Value{}, fmt.Errorf("di: circular dependency on %v", typeFullName(t))
	}
	visiting[t] = true
	defer delete(visiting, t)

	c.mu.RLock()
	inst, ok := c.instances[t]
	c.mu.RUnlock()
	if ok {
		return inst, nil
	}

	c.mu.RLock()
	ctor, ok := c.providers[t]
	c.mu.RUnlock()
	if !ok {
		return reflect.Value{}, fmt.Errorf("di: no provider for type %v", typeFullName(t))
	}

	ctorType := ctor.Type()
	numIn := ctorType.NumIn()
	args := make([]reflect.Value, numIn)
	for i := range numIn {
		argType := ctorType.In(i)
		argVal, err := c.resolve(argType, visiting)
		if err != nil {
			return reflect.Value{}, err
		}
		args[i] = argVal
	}

	results := ctor.Call(args)
	var val reflect.Value
	var err error
	if len(results) == 1 {
		val = results[0]
	} else if len(results) == 2 {
		val = results[0]
		if e := results[1]; !e.IsNil() {
			if e.Type().Implements(reflect.TypeFor[error]()) {
				err = e.Interface().(error)
			} else {
				err = fmt.Errorf("di: second return value is not error")
			}
		}
	} else {
		return reflect.Value{}, fmt.Errorf("di: constructor returned %d values, expected 1 or 2", len(results))
	}
	if err != nil {
		return reflect.Value{}, err
	}

	c.mu.Lock()
	c.instances[t] = val
	c.mu.Unlock()
	return val, nil
}

// Run 执行所有延迟的 Invoke 任务
func (c *container) Run(ctx context.Context) error {
	for i, fn := range c.pending {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := fn(); err != nil {
			return fmt.Errorf("di: invoke task %d failed: %w", i, err)
		}
	}
	return nil
}

func typeFullName(t reflect.Type) string {
	if t == nil {
		return "<nil>"
	}
	// 内置类型（如 string、int）无包路径，直接输出类型名
	if t.PkgPath() == "" {
		return t.Name()
	}
	return t.PkgPath() + "." + t.Name()
}
