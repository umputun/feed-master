package errs

import (
	"errors"
	"fmt"
	"strings"
)

// ArithmeticError // любая ошибка связанная с арифметикой
// AssertionError // ошибка приведения типов
// IndexError // индекс в слайсе не неайден
// KeyError // ключ в мапе не найден
// NameError
// TypeError // когда в интерфейсе передан неправильный тип
// InvalidTypeError
// EnumError
// NotUniqueError
// TooManyParamsError
// TooFewParamsError
// ForbiddenError

type NotImplementedError struct {
	Name string
}

func NotImplemented(a ...string) *NotImplementedError {
	return &NotImplementedError{strings.Join(a, "; ")}
}

func (e *NotImplementedError) Error() string {
	r := "not implemented"
	if e.Name != "" {
		r = e.Name + ": " + r
	}
	return r
}

// RecursionError
// RuntimeError
// IOError
// EnvironmentError // любая ошибка в конфигурации

// что-то не существует
type NotFoundError struct {
	Type string
	Key  string
}

func NotFound(_type, key string) *NotFoundError {
	return &NotFoundError{_type, key}
}

func (err *NotFoundError) Error() string {
	spacer := "'"
	if err.Type != "" {
		spacer = " '"
	}
	return err.Type + spacer + err.Key + "' not found"
}

func IsNotFound(err error) bool {
	_, ok := err.(*NotFoundError)
	return ok
}

type NotPointerError struct {
	VariableName string
}

func NotPointer(varName string) *NotPointerError {
	return &NotPointerError{varName}
}

func (err *NotPointerError) Error() string {
	return err.VariableName + " is not a pointer"
}

// ошибка доступа. как к файлу, так и к чему либо другому
type PermissionError struct {
	Item          string
	RequiredScope string
}

func Permission(item string) *PermissionError {
	return &PermissionError{Item: item}
}

func (err *PermissionError) Scope(name string) *PermissionError {
	err.RequiredScope = name
	return err
}

func (err *PermissionError) Error() string {
	res := err.Item + ": permission denied"
	if err.RequiredScope != "" {
		res += " (required " + err.RequiredScope + " scope)"
	}
	return res
}

func IsPermission(err error) bool {
	_, ok := err.(*PermissionError)
	return ok
}

// TimeoutError // таймаут, можно использовать вкупе с context

// когда много ошибок, о которых надо сообщить
type MultipleErrors struct {
	errs []error
}

func Multiple(es ...error) *MultipleErrors {
	return &MultipleErrors{es}
}

// DEPRECATED: use it only when you changing legacy code only
func MultipleAsString(es ...string) *MultipleErrors {
	res := make([]error, len(es))
	for i, err := range es {
		res[i] = errors.New(err)
	}
	return &MultipleErrors{res}
}

func (err *MultipleErrors) Add(errs ...error) {
	if err.errs == nil {
		err.errs = make([]error, 0)
	}

	for _, e := range errs {
		if e == nil {
			continue
		}

		err.errs = append(err.errs, e)
	}
}

// приводит ошибку к общепризнаному виду
func (err *MultipleErrors) Normalize() error {
	switch len(err.errs) {
	case 0:
		return nil
	case 1:
		return err.errs[0]
	default:
		return err
	}
}

func (err *MultipleErrors) Errors() []error {
	return err.errs
}

func (err *MultipleErrors) Error() string {
	switch len(err.errs) {
	case 0:
		return "(0 errors)"
	case 1:
		return err.errs[0].Error()
	case 2:
		return err.errs[0].Error() + " (and 1 other error)"
	default:
		return fmt.Sprintf("%s (and %d other errors)",
			err.errs[0].Error(), len(err.errs)-1)
	}
}

func IsMultiple(err error) bool {
	_, ok := err.(*MultipleErrors)
	return ok
}
