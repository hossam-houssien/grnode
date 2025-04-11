package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

// 1.  Simplified struct to hold node data.
type NodeData struct {
	Name     string
	Path     string
	Synopsis string
	URL      string
}

// 2. Struct for graph metadata.
type GraphMetadata struct {
	Name        string
	BackgroundColor string
	FontName    string
}

// 3.  Struct to hold edge data.
type EdgeData struct {
	From     string
	To       string
	Relation string // Optional relation name
	Color    string
	Style    string
}

// 4. Config struct to hold all configuration.
type Config struct {
	Graph       GraphMetadata
	NodeFormat  string
	EdgeFormat  string
	AvailableGraphAttributes    []string
	AvailableNodeAttributes     []string
	AvailableEdgeAttributes     []string
}

// 5. Constants
const (
	version = "0.1.0" // Start versioning
	defaultConfigTemplate = `
# Default Configuration for Graphviz Generator

# Graph Metadata
[graph]
name = "MyGraph"
# background_color = "lightgray" # Optional
# font_name = "Arial" # Optional

# Node Data File Format
# Example: name|path|synopsis|url
node_format = "name|path|synopsis|url"

# Edge Data File Format
# Example: from_name,to_name[,relation,color,style]
edge_format = "from_name,to_name[,relation,color,style]"

# Available Graph Attributes in the config file
# These can be set in the [graph] section
available_graph_attributes = ["bgcolor", "fontname"]

# Available Node Attributes in the node data file
available_node_attributes = ["name", "path", "synopsis", "url"]

# Available Edge Attributes in the edge data file
# These can be used in the edge_format
available_edge_attributes = ["relation", "color", "style"]
`
)

func renderGraph(graphMeta GraphMetadata, nodes []NodeData, edges []EdgeData, dotOutput string) ([]byte, error) {
	var in, out bytes.Buffer

	// Graph styling
	fmt.Fprintf(&in, "digraph %s { \n", graphMeta.Name)
	if graphMeta.BackgroundColor != "" {
		fmt.Fprintf(&in, "  bgcolor=\"%s\";\n", graphMeta.BackgroundColor)
	}
	if graphMeta.FontName != "" {
		fmt.Fprintf(&in, "  fontname=\"%s\";\n", graphMeta.FontName)
	}
	// Default node styling
	fmt.Fprintf(&in, "  node [shape=box, style=filled, fillcolor=\"#e0e0e0\", fontname=\"Arial\"];\n")
	// Default edge styling
	fmt.Fprintf(&in, "  edge [color=\"#555555\", fontname=\"Arial\"];\n")

	for i, node := range nodes {
		fmt.Fprintf(&in, "  n%d [label=\"%s\", URL=\"%s\", tooltip=\"%s\"];\n",
			i, node.Name, node.URL,
			strings.Replace(node.Synopsis, `"`, `\"`, -1))
	}

	// Create a map to look up node indices by name.
	nodeIndexMap := make(map[string]int)
	for i, node := range nodes {
		nodeIndexMap[node.Name] = i
	}

	for _, edge := range edges {
		fromIndex, fromFound := nodeIndexMap[edge.From]
		toIndex, toFound := nodeIndexMap[edge.To]
		if !fromFound || !toFound {
			return nil, fmt.Errorf("error: edge refers to unknown node(s) from: %s, to: %s", edge.From, edge.To)
		}
		fmt.Fprintf(&in, "  n%d -> n%d", fromIndex, toIndex)
		hasAttributes := false
		if edge.Relation != "" {
			fmt.Fprintf(&in, " [label=\"%s\"", edge.Relation)
			hasAttributes = true
		}
		if edge.Color != "" {
			if !hasAttributes {
				fmt.Fprintf(&in, " [color=\"%s\"", edge.Color)
				hasAttributes = true
			} else {
				fmt.Fprintf(&in, ", color=\"%s\"", edge.Color)
			}

		}
		if edge.Style != "" {
			if !hasAttributes {
				fmt.Fprintf(&in, " [style=\"%s\"", edge.Style)
			} else {
				fmt.Fprintf(&in, ", style=\"%s\"", edge.Style)
			}
		}
		if hasAttributes {
			fmt.Fprintf(&in, "]")
		}
		fmt.Fprintf(&in, ";\n")
	}
	in.WriteString("}")

	if dotOutput != "" {
		err := os.WriteFile(dotOutput, in.Bytes(), 0644)
		if err != nil {
			return nil, fmt.Errorf("error writing DOT file: %v", err)
		}
		fmt.Printf("DOT output written to %s\n", dotOutput)
	}

	cmd := exec.Command("dot", "-Tsvg")
	cmd.Stdin = &in
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	p := out.Bytes()
	i := bytes.Index(p, []byte("<svg"))
	if i < 0 {
		return nil, errors.New("<svg not found")
	}
	p = p[i:]
	return p, nil
}

func main() {
	// 6. Define command-line flags
	nodesFile := flag.String("nodes", "nodes.txt", "File containing node information")
	edgesFile := flag.String("edges", "edges.txt", "File containing edge information")
	outputFile := flag.String("output", "graph.svg", "File to write the SVG output")
	graphName := flag.String("name", "MyGraph", "Name of the graph")
	graphBgColor := flag.String("bgcolor", "", "Background color of the graph")
	graphFontName := flag.String("fontname", "", "Font name for the graph")
	dotOutputFile := flag.String("dot", "", "File to write the DOT output for debugging")
	generateConfig := flag.Bool("genconfig", false, "Generate a default configuration file")
	versionFlag := flag.Bool("version", false, "Show version and exit") // New version flag

	flag.Parse()

	if *versionFlag {
		fmt.Printf("Graphviz Graph Generator version %s\n", version)
		os.Exit(0)
	}

	if *generateConfig {
		tmpl, err := template.New("config").Parse(defaultConfigTemplate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing config template: %v\n", err)
			os.Exit(1)
		}
		err = tmpl.Execute(os.Stdout, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error executing config template: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("\nDefault configuration written to standard output.  Save to config.toml and edit.")
		os.Exit(0)
	}

	// 7. Read node data from file
	nodeLines, err := os.ReadFile(*nodesFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading nodes file: %v\n", err)
		os.Exit(1)
	}
	nodeStrings := strings.Split(string(nodeLines), "\n")
	nodes := make([]NodeData, 0, len(nodeStrings))
	for _, line := range nodeStrings {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) != 4 {
			fmt.Fprintf(os.Stderr, "Error: Invalid node format in %s: %s (expected 'name|path|synopsis|url')\n", *nodesFile, line)
			os.Exit(1)
		}
		nodes = append(nodes, NodeData{
			Name:     strings.TrimSpace(parts[0]),
			Path:     strings.TrimSpace(parts[1]),
			Synopsis: strings.TrimSpace(parts[2]),
			URL:      strings.TrimSpace(parts[3]),
		})
	}

	// 8. Read edge data from file
	edgeLines, err := os.ReadFile(*edgesFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading edges file: %v\n", err)
		os.Exit(1)
	}
	edgeStrings := strings.Split(string(edgeLines), "\n")
	edges := make([]EdgeData, 0, len(edgeStrings))
	for _, line := range edgeStrings {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 2 { // Changed to allow edges with only from,to
			fmt.Fprintf(os.Stderr, "Error: Invalid edge format in %s: %s (expected 'from_name,to_name[,relation,color,style]')\n", *edgesFile, line)
			os.Exit(1)
		}
		edgeData := EdgeData{
			From:  strings.TrimSpace(parts[0]),
			To:    strings.TrimSpace(parts[1]),
		}
		if len(parts) > 2 && strings.TrimSpace(parts[2]) != "" {
			edgeData.Relation = strings.TrimSpace(parts[2])
		}
		if len(parts) > 3 && strings.TrimSpace(parts[3]) != "" {
			edgeData.Color = strings.TrimSpace(parts[3])
		}
		if len(parts) > 4 && strings.TrimSpace(parts[4]) != "" {
			edgeData.Style = strings.TrimSpace(parts[4])
		}
		edges = append(edges, edgeData)
	}

	// 9. Call the renderGraph function
	graphMeta := GraphMetadata{
		Name:            *graphName,
		BackgroundColor: *graphBgColor,
		FontName:        *graphFontName,
	}
	svgData, err := renderGraph(graphMeta, nodes, edges, *dotOutputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating graph: %v\n", err)
		os.Exit(1)
	}

	// 10. Write the SVG data to the output file
	err = os.WriteFile(*outputFile, svgData, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing to %s: %v\n", *outputFile, err)
		os.Exit(1)
	}

	fmt.Printf("Successfully wrote graph SVG to %s\n", *outputFile)
}

