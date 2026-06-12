package di

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
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

// Supply 直接注册一个已存在的实例（单例）
func (c *container) Supply(value any) {
	t := reflect.TypeOf(value)
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.instances[t]; ok {
		panicf("di: instance already supplied for type %v", typeFullName(t))
	}
	if _, ok := c.providers[t]; ok {
		panicf("di: provider already exists for type %v, cannot supply", typeFullName(t))
	}
	c.instances[t] = reflect.ValueOf(value)
}

// Provide 注册构造函数，支持 func() T 或 func() (T, error)
func (c *container) Provide(constructor any) {
	ctorType := reflect.TypeOf(constructor)
	if ctorType.Kind() != reflect.Func {
		panicf("di: constructor must be a function, got %s", typeFullName(ctorType))
	}
	numOut := ctorType.NumOut()
	if numOut != 1 && numOut != 2 {
		panicf("di: constructor must return 1 or 2 values, got %d", numOut)
	}
	if numOut == 2 {
		if !ctorType.Out(1).Implements(reflect.TypeFor[error]()) {
			panicf("di: second return value must be error, got %v", typeFullName(ctorType.Out(1)))
		}
	}
	returnType := ctorType.Out(0)

	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.providers[returnType]; ok {
		panicf("di: already registered for type %v", typeFullName(returnType))
	}
	c.providers[returnType] = reflect.ValueOf(constructor)
}

// Invoke 注册一个延迟执行的函数，参数自动注入，函数可返回 error
func (c *container) Invoke(fn any) {
	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func {
		panicf("di: Invoke requires a function, got %s", typeFullName(fnType))
	}
	wrapper := func() error {
		fnVal := reflect.ValueOf(fn)
		numIn := fnType.NumIn()
		args := make([]reflect.Value, numIn)
		for i := range numIn {
			argType := fnType.In(i)
			argVal, err := c.resolve(argType, map[reflect.Type]bool{}, nil)
			if err != nil {
				return errorf("di: cannot resolve argument %d (%v): %w", i, typeFullName(argType), err)
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

// getFuncName 从 reflect.Value 获取函数名（辅助函数）
func getFuncName(fn reflect.Value) string {
	if fn.Kind() != reflect.Func {
		return "<not a function>"
	}
	pc := fn.Pointer()
	if pc == 0 {
		return "<nil function>"
	}
	fnPtr := runtime.FuncForPC(pc)
	if fnPtr == nil {
		return "<unknown>"
	}
	return fnPtr.Name()
}

// resolve 递归解析类型，支持 (T, error) 构造函数
func (c *container) resolve(t reflect.Type, visiting map[reflect.Type]bool, path []reflect.Type) (reflect.Value, error) {
	if visiting[t] {
		return reflect.Value{}, errorf("di: circular dependency on %v", typeFullName(t))
	}
	visiting[t] = true
	defer delete(visiting, t)

	currentPath := append(path, t)

	// 优先返回已缓存的实例
	c.mu.RLock()
	inst, ok := c.instances[t]
	c.mu.RUnlock()
	if ok {
		return inst, nil
	}

	// 查找构造函数
	c.mu.RLock()
	ctor, ok := c.providers[t]
	c.mu.RUnlock()
	if !ok {
		pathStr := buildPathString(currentPath)
		return reflect.Value{}, errorf("di: no provider for type %v\n\trequired by: %s", typeFullName(t), pathStr)
	}

	ctorType := ctor.Type()
	numIn := ctorType.NumIn()
	args := make([]reflect.Value, numIn)

	// 获取构造函数名称用于错误信息（关键修改1）
	ctorName := getFuncName(ctor)

	for i := range numIn {
		argType := ctorType.In(i)
		argVal, err := c.resolve(argType, visiting, currentPath)
		if err != nil {
			// 关键修改2：在错误中附上构造函数名称和参数索引
			return reflect.Value{}, errorf("di: in constructor %s: cannot resolve argument %d (%v): %w",
				ctorName, i, typeFullName(argType), err)
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
		if e := results[1]; !e.IsNil() && e.Type().Implements(reflect.TypeFor[error]()) {
			err = e.Interface().(error)
		}
	} else {
		return reflect.Value{}, errorf("di: constructor returned %d values, expected 1 or 2", len(results))
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
			return errorf("di: invoke task %d failed: %w", i, err)
		}
	}
	return nil
}

func typeFullName(t reflect.Type) string {
	if t == nil {
		return "<nil>"
	}
	raw := t.String()
	elemT := t
	for elemT.Kind() == reflect.Pointer {
		elemT = elemT.Elem()
	}
	if elemT.PkgPath() == "" {
		return raw
	}
	full := elemT.PkgPath() + "." + elemT.Name()
	idx := len(raw) - len(elemT.Name())
	return raw[:idx] + full
}

func panicf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	stack := debug.Stack()
	panic(fmt.Sprintf("%s\n\n%s", msg, stack))
}

func errorf(format string, args ...any) error {
	baseErr := fmt.Errorf(format, args...)
	pc, file, line, ok := runtime.Caller(2)
	if !ok {
		return baseErr
	}
	funcName := runtime.FuncForPC(pc).Name()
	return fmt.Errorf("%w\n    at %s:%d in %s", baseErr, file, line, funcName)
}

func buildPathString(types []reflect.Type) string {
	if len(types) == 0 {
		return ""
	}
	var b strings.Builder
	for i, t := range types {
		if i > 0 {
			b.WriteString(" -> ")
		}
		b.WriteString(typeFullName(t))
	}
	return b.String()
}
