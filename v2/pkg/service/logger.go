package service

import "io"

type Logger interface {
	Printf(format string, v ...any)
	Print(v ...any)
	Println(v ...any)
	Fatal(v ...any)
	Fatalf(format string, v ...any)
	Fatalln(v ...any)
	Panic(v ...any)
	Panicf(format string, v ...any)
	Panicln(v ...any)
	Output(calldepth int, s string) error
	Writer() io.Writer
	SetPrefix(prefix string)
	Prefix() string
	SetFlags(flag int)
	Flags() int
	SetOutput(w io.Writer)
}
