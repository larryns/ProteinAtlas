package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
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

func main() {
	// Open our xmlFile
	xmlFile, err := os.Open("/Users/singhln/Downloads/ENSG00000078328.xml")
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}

	// defer the closing of our xmlFile so that we can parse it later on
	defer xmlFile.Close()

	// argument flags, read in a file, default called genes.txt
	wordPtr := flag.String("file", "genes.txt", "input file name")
	flag.Parse()

	// read a file of symbols and return a symbol string
	symbols := readSymbols(wordPtr)

	// The data to pass gene symbols to the REST api. Values is a map indexed by string to a list of strings.
	data := url.Values{"symbols": symbols}

	// Marshal performs a JSON encoding of data.
	requestBody, err := json.Marshal(data)
	die(err)

	// Create a client for the REST API.
	resp, err := http.Post("https://rest.ensembl.org/lookup/symbol/homo_sapiens", "application/json", bytes.NewBuffer(requestBody))
	die(err)
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	die(err)

	// Convert the json object back to a map. Ensembl returns a map of the symbols you search for
	// indexed by the symbol name. Each symbol name points to a map of information indexed by the
	// type of information, e.g. id, version, assembly_name, etc. This map can point to either a string
	// or number, hence the use of interface.
	type GeneInfo map[string]map[string]interface{}
	geneList := GeneInfo{}
	err = json.Unmarshal(body, &geneList)
	die(err)

	// Go through each gene and get the protein atlas information
	for symbol, info := range geneList {
		protAtlas(symbol, (info["id"]).(string)) // Need a type assertion to convert interface{} to string
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
	fmt.Println(entry.Entry.Name)
	fmt.Printf("\tTissue specificity: ")
	for _, e := range entry.Entry.RNAExpr {
		if e.AssayType == "consensusTissue" {
			fmt.Println(strings.Join(e.RNASpecificity.Tissue, ", "))
		}
	}
	fmt.Printf("\tSingle cell type specificity: ")
	fmt.Println(strings.Join(entry.Entry.CellTypeExpr.CellTypeSpecificity.CellType, ","))
	fmt.Printf("\tSingle cell type expression cluster: ")
	fmt.Println(entry.Entry.CellTypeExpr.CellTypeExprCluster)
	fmt.Printf("\tImmune cell specificity: ")
	for _, e := range entry.Entry.RNAExpr {
		if e.AssayType == "immuneCell" {
			fmt.Println(e.RNASpecificity.Specificity)
		}
	}
	fmt.Printf("\tBrain specificity: ")
	for _, e := range entry.Entry.RNAExpr {
		if e.AssayType == "humanBrainRegional" {
			fmt.Println(e.RNASpecificity.Specificity)
		}
	}
}
