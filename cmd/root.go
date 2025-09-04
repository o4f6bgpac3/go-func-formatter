package cmd

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

type Method struct {
	decl  *ast.FuncDecl
	start token.Pos
	end   token.Pos
}

type ByName []Method

func (m ByName) Len() int           { return len(m) }
func (m ByName) Swap(i, j int)      { m[i], m[j] = m[j], m[i] }
func (m ByName) Less(i, j int) bool { return m[i].decl.Name.Name < m[j].decl.Name.Name }

type ByPos []Method

func (m ByPos) Len() int           { return len(m) }
func (m ByPos) Swap(i, j int)      { m[i], m[j] = m[j], m[i] }
func (m ByPos) Less(i, j int) bool { return m[i].start < m[j].start }

var rootCmd = &cobra.Command{
	Use:   "reordertool [file]",
	Short: "Reorders Go methods in a file alphabetically by name",
	Args:  cobra.ExactArgs(1),
	RunE:  run,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	inputFile := args[0]

	src, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", inputFile, err)
	}

	fSet := token.NewFileSet()
	file, err := parser.ParseFile(fSet, inputFile, src, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse file %s: %w", inputFile, err)
	}

	// Separate methods vs others
	var methods []Method

	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if funcDecl.Recv == nil {
			continue
		}

		// Exclude constructors or funcs starting with "New"
		if strings.HasPrefix(funcDecl.Name.Name, "New") && funcDecl.Recv.NumFields() > 0 {
			continue
		}

		start := funcDecl.Pos()
		if funcDecl.Doc != nil {
			start = funcDecl.Doc.Pos()
		}
		end := funcDecl.End()

		methods = append(methods, Method{decl: funcDecl, start: start, end: end})
	}

	if len(methods) == 0 {
		fmt.Printf("No methods to reorder\n")
		return nil
	}

	// Sort methods alphabetically by name
	sort.Sort(ByName(methods))

	// To get the block, sort by position to find first and last
	posMethods := append([]Method(nil), methods...)
	sort.Sort(ByPos(posMethods))
	firstStartOff := fSet.Position(posMethods[0].start).Offset
	lastEndOff := fSet.Position(posMethods[len(posMethods)-1].end).Offset

	// Get sorted sources
	var sortedSources []string
	for _, m := range methods {
		startOff := fSet.Position(m.start).Offset
		endOff := fSet.Position(m.end).Offset
		sortedSources = append(sortedSources, string(src[startOff:endOff]))
	}

	joined := strings.Join(sortedSources, "\n\n")

	// Build new source
	var newSrc bytes.Buffer
	newSrc.Write(src[0:firstStartOff])
	newSrc.WriteString(joined)
	newSrc.Write(src[lastEndOff:])

	// Write output.txt
	//outputFile := "output.txt"
	//if err := os.WriteFile(outputFile, newSrc.Bytes(), 0644); err != nil {
	//	return fmt.Errorf("failed to write output file %s: %w", outputFile, err)
	//}

	//fmt.Printf("Methods reordered and written to %s\n", outputFile)

	// Open output.txt in TextEdit (macOS)
	//err = exec.Command("open", "-a", "TextEdit", outputFile).Start()
	//if err != nil {
	//	return fmt.Errorf("failed to open %s in TextEdit: %w", outputFile, err)
	//}

	if err := os.WriteFile(inputFile, newSrc.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write to file %s: %w", inputFile, err)
	}

	fmt.Printf("Methods reordered in %s\n", inputFile)

	return nil
}
