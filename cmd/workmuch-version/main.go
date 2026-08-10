package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"workmuch-go/internal/versioncalc"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) > 0 && args[0] == "validate-tag" {
		if len(args) != 2 {
			return fmt.Errorf("usage: workmuch-version validate-tag <tag>")
		}
		result, err := versioncalc.ValidateTag(context.Background(), ".", args[1])
		if err != nil {
			return err
		}
		fmt.Println(result.Version)
		return nil
	}

	flags := flag.NewFlagSet("workmuch-version", flag.ContinueOnError)
	format := flags.String("format", "version", "output field: version, tag, base, head, main-ref, plist-short, or plist-build")
	head := flags.String("head", "", "release commit to calculate (default HEAD)")
	mainBase := flags.String("main-base", "", "recorded main merge-base")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}

	result, err := versioncalc.Calculate(context.Background(), ".", versioncalc.Options{
		Head:     *head,
		MainBase: *mainBase,
	})
	if err != nil {
		return err
	}

	switch *format {
	case "version":
		fmt.Println(result.Version)
	case "tag":
		fmt.Println(result.Tag)
	case "base":
		fmt.Println(result.MainBase)
	case "head":
		fmt.Println(result.Head)
	case "main-ref":
		fmt.Println(result.MainRef)
	case "plist-short", "plist-build":
		versions, parseErr := versioncalc.ParsePlistVersions(result.Version)
		if parseErr != nil {
			return parseErr
		}
		if *format == "plist-short" {
			fmt.Println(versions.Short)
		} else {
			fmt.Println(versions.Build)
		}
	default:
		return fmt.Errorf("unsupported output format %q", *format)
	}
	return nil
}
