package main

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

// Type for extracting the data we need. Struct fields mus tstart with upper case letter (exported)
type ProteinAtlas struct {
	Entry Entry `xml:"entry"`
}

type Entry struct {
	Name         string             `xml:"name"`
	RNAExpr      []RNAExpression    `xml:"rnaExpression"`
	CellTypeExpr CellTypeExpression `xml:"cellTypeExpression"`
}

type RNAExpression struct {
	AssayType      string         `xml:"assayType,attr"`
	RNASpecificity RNASpecificity `xml:"rnaSpecificity"`
}

type RNASpecificity struct {
	Tissue      []string `xml:"tissue"`
	Specificity string   `xml:"specificity,attr"`
}

type CellTypeExpression struct {
	CellTypeSpecificity CellTypeSpecificity `xml:"cellTypeSpecificity"`
	CellTypeExprCluster string              `xml:"cellTypeExpressionCluster"`
}

type CellTypeSpecificity struct {
	CellType []string `xml:"cellType"`
}

type Response struct {
	Data []Orthologue `json:"data"`
}

type Orthologue struct {
	Homologies []Homologies `json:"homologies"`
}

type Homologies struct {
	Target Target `json:"target"`
}

type Target struct {
	Id string `json:"id"`
}

func main() {
	// argument flags, read in a file, default called genes.txt
	wordPtr := flag.String("file", "genes.txt", "input file name")
	flag.Parse()

	// read a file of symbols and return a symbol string
	symbols := readSymbols(wordPtr)

	// Print the header
	fmt.Println("Id\tSymbol\tTissue Specificity\tSingle cell type specificity\tSingle cell type expression cluster\tImmune cell specificity\tBrain specificity")

	// Create a client for the REST API.
	for _, id := range symbols {
		if len(id) == 0 {
			continue
		}

		resp, err := http.Get("https://rest.ensembl.org/homology/symbol/Sus_scrofa/" + id + "?sequence=cdna;target_taxon=9606;target_species=human;type=orthologues;content-type=application/json")
		die(err)
		defer resp.Body.Close()

		data, err := ioutil.ReadAll(resp.Body)
		die(err)

		var orthologue Response
		json.Unmarshal(data, &orthologue)

		fmt.Print(id)
		for _, data := range orthologue.Data {
			for _, homology := range data.Homologies {
				protAtlas(id, homology.Target.Id)
			}
		}
		fmt.Print("\n")
	}

}

// Utility function for checking error codes
func die(e error) {
	if e != nil {
		panic(e)
	}
}

// Convert a file of symbols to a string: "symbols" : { <gene list> }
func readSymbols(fileName *string) []string {
	file, err := os.Open(*fileName)
	die(err)
	defer file.Close()

	// Our array of strings
	var lines []string

	// create a scanner to read the file
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines
}

// protein atlas call to get tissue specific information.
func protAtlas(symbol string, id string) {
	// we're using get this time to get the protein atlas information.
	resp, err := http.Get("https://proteinatlas.org/" + id + ".xml")
	die(err)
	defer resp.Body.Close()

	dat, err := ioutil.ReadAll(resp.Body)
	die(err)

	// Parse the XML
	var entry ProteinAtlas
	xml.Unmarshal(dat, &entry)

	// Now format and print the output
	// First print the header
	fmt.Print("\t" + entry.Entry.Name + "\t")
	for _, e := range entry.Entry.RNAExpr {
		if e.AssayType == "consensusTissue" {
			fmt.Print(strings.Join(e.RNASpecificity.Tissue, ", "))
		}
	}
	fmt.Print("\t" + strings.Join(entry.Entry.CellTypeExpr.CellTypeSpecificity.CellType, ","))
	fmt.Print("\t" + entry.Entry.CellTypeExpr.CellTypeExprCluster)
	for _, e := range entry.Entry.RNAExpr {
		if e.AssayType == "immuneCell" {
			fmt.Print("\t" + e.RNASpecificity.Specificity)
		}
	}
	fmt.Print("\t")
	for _, e := range entry.Entry.RNAExpr {
		if e.AssayType == "humanBrainRegional" {
			fmt.Print(e.RNASpecificity.Specificity)
		}
	}
}
