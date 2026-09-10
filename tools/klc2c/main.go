// Command klc2c generates a Windows keyboard layout DLL's C source (the
// KBDTABLES / KbdLayerDescriptor definition) from a .klc file, without
// needing MSKLC's kbdutool.exe.
//
// Usage:
//
//	klc2c -in layout.klc -out layout.c [-def layout.def]
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	in := flag.String("in", "", "input .klc file (required)")
	out := flag.String("out", "", "output .c file (required)")
	defOut := flag.String("def", "", "output .def file (optional)")
	flag.Parse()

	if *in == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "usage: klc2c -in layout.klc -out layout.c [-def layout.def]")
		os.Exit(2)
	}

	layout, err := ParseKLC(*in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "klc2c: parsing %s: %v\n", *in, err)
		os.Exit(1)
	}

	src, err := Generate(layout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "klc2c: generating C source: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*out, []byte(src), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "klc2c: writing %s: %v\n", *out, err)
		os.Exit(1)
	}
	fmt.Printf("klc2c: wrote %s (%d LAYOUT rows, %d shift states)\n", *out, len(layout.Rows), len(layout.ShiftStates))

	if *defOut != "" {
		if err := os.WriteFile(*defOut, []byte(GenerateDef(layout)), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "klc2c: writing %s: %v\n", *defOut, err)
			os.Exit(1)
		}
		fmt.Printf("klc2c: wrote %s\n", *defOut)
	}
}
