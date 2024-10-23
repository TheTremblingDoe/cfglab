package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
)

func main() {
	// Open the input file
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "input.go", nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	// Find the function declaration
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		// Generate the control flow graph
		cfg := generateCFG(funcDecl)

		// Write the CFG to an output file in DOT format
		f, err := os.Create("output.dot")
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		fmt.Fprintln(f, "digraph CFG {")
		for _, node := range cfg.Nodes {
			// Assign shapes based on node kind
			shape := "box" // default shape
			if node.Kind == "entry" {
				shape = "diamond"
			}
			fmt.Fprintf(f, "  %s [label=\"%s\", shape=\"%s\"];\n", getNodeID(node), getSourceString(node.Stmt, fset), shape)
		}
		for _, node := range cfg.Nodes {
			for _, edge := range node.Edges {
				fmt.Fprintf(f, "  %s -> %s;\n", getNodeID(node), getNodeIDByStmt(edge.Stmt, cfg.Nodes))
			}
		}
		fmt.Fprintln(f, "}")
	}
}

func getNodeID(node *CFGNode) string {
	if node.Stmt == nil {
		return "entry"
	}
	return fmt.Sprintf("node%d", node.Stmt.Pos())
}

func getNodeIDByStmt(stmt ast.Stmt, nodes []*CFGNode) string {
	for _, node := range nodes {
		if node.Stmt == stmt {
			return getNodeID(node)
		}
	}
	return ""
}

type CFGNode struct {
	Stmt  ast.Stmt
	Kind  string
	Edges []*CFGEdge
}

type CFGEdge struct {
	Stmt ast.Stmt
	Kind string
}

type CFG struct {
	Nodes []*CFGNode
}

func generateCFG(funcDecl *ast.FuncDecl) *CFG {
	cfg := &CFG{Nodes: []*CFGNode{}}
	nodeMap := make(map[ast.Stmt]*CFGNode)

	// Create a node for the function entry point
	entryNode := &CFGNode{Stmt: nil, Kind: "entry"}
	cfg.Nodes = append(cfg.Nodes, entryNode)
	nodeMap[entryNode.Stmt] = entryNode

	// Create nodes for each statement in the function body
	for _, stmt := range funcDecl.Body.List {
		createCFGNode(stmt, entryNode, cfg, nodeMap)
	}

	return cfg
}

func createCFGNode(stmt ast.Stmt, parentNode *CFGNode, cfg *CFG, nodeMap map[ast.Stmt]*CFGNode) {
	var node *CFGNode
	switch stmt := stmt.(type) {
	case *ast.ExprStmt:
		node = &CFGNode{Stmt: stmt, Kind: "expr"}
	case *ast.ReturnStmt:
		node = &CFGNode{Stmt: stmt, Kind: "return"}
	case *ast.IfStmt:
		node = &CFGNode{Stmt: stmt, Kind: "if"}
		// Create nodes for the if statement's branches
		for _, branch := range stmt.Body.List {
			createCFGNode(branch, node, cfg, nodeMap)
		}
		// Handle the else branch if present
		if stmt.Else != nil {
			createCFGNode(stmt.Else, node, cfg, nodeMap)
		}
	case *ast.ForStmt:
		node = &CFGNode{Stmt: stmt, Kind: "for"}
		// Create nodes for the loop body
		for _, bodyStmt := range stmt.Body.List {
			createCFGNode(bodyStmt, node, cfg, nodeMap)
		}
		// Add back edge for the loop
		node.Edges = append(node.Edges, &CFGEdge{Stmt: node.Stmt, Kind: "loop"})
	case *ast.RangeStmt:
		node = &CFGNode{Stmt: stmt, Kind: "range"}
		// Create nodes for the loop body
		for _, bodyStmt := range stmt.Body.List {
			createCFGNode(bodyStmt, node, cfg, nodeMap)
		}
		// Add back edge for the range loop
		node.Edges = append(node.Edges, &CFGEdge{Stmt: node.Stmt, Kind: "loop"})
	default:
		log.Printf("unsupported statement type: %T", stmt)
		return
	}

	// Append the node to the CFG and create the edge
	cfg.Nodes = append(cfg.Nodes, node)
	nodeMap[stmt] = node
	parentNode.Edges = append(parentNode.Edges, &CFGEdge{Stmt: stmt, Kind: "next"})
}

func getSourceString(stmt ast.Stmt, fset *token.FileSet) string {
	if stmt == nil {
		return ""
	}

	var endPos int

	pos := stmt.Pos()
	file := fset.File(pos)
	line := file.Line(pos)
	startPos := file.LineStart(line)
	fileContent, _ := os.ReadFile(file.Name())
	startOffset := int(file.Offset(startPos))

	for i := startOffset; i < len(fileContent); i++ {
		if fileContent[i] == '\n' {
			endPos = i
			break
		}
	}

	if endPos == 0 {
		endPos = len(fileContent)
	}

	lineBytes := fileContent[startOffset:endPos]

	return string(lineBytes)
}
