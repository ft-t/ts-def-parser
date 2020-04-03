package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"
)

type Trailers struct {
	Variants    []string `json:"variants"`
	Definitions map[string]TrailerDefinition `json:"definitions"`
}

type TrailerDefinition struct {
	Countries []string `json:"countries"`
}

type Company struct {
	Cities     []string `json:"cities"`
	CargoesIn  []string `json:"cargoes_in"`
	CargoesOut []string `json:"cargoes_out"`
}

type City struct {
	Country string `json:"country"`
}

var trailerDefRegex *regexp.Regexp
var trailerDefCountryRegex *regexp.Regexp
var companyCityRegex *regexp.Regexp
var companyCargoRegex *regexp.Regexp

func main() {
	baseDir := flag.String("base-dir", ".", "directory with unpacked def files")
	dlcName := flag.String("dlc", "base_game", "dlc name, will be added to resulted files")
	flag.Parse()

	trailerDir := path.Join(*baseDir, "vehicle", "trailer")
	trailerDefDir := path.Join(*baseDir, "vehicle", "trailer_defs")
	cargoDir := path.Join(*baseDir, "cargo")
	companyDir := path.Join(*baseDir, "company")
	cityDir := path.Join(*baseDir, "city")

	trailerFiles, _ := ioutil.ReadDir(trailerDir)
	trailerDefFiles, _ := ioutil.ReadDir(trailerDefDir)
	cargoFiles, _ := ioutil.ReadDir(cargoDir)
	companyFiles, _ := ioutil.ReadDir(companyDir)
	cityFiles, _ := ioutil.ReadDir(cityDir)

	trailerRegex, _ := regexp.Compile("^\\s*trailer\\s*:")
	trailerDefRegex, _ = regexp.Compile("^\\s*trailer_def\\s*:")
	trailerDefCountryRegex, _ = regexp.Compile("^\\s*country_validity\\[]\\s*:")
	cargoRegex, _ := regexp.Compile("^\\s*cargo_data\\s*:")
	companyCityRegex, _ = regexp.Compile("^\\s*company_def\\s*:")
	companyCargoRegex, _ = regexp.Compile("^\\s*cargo_def\\s*:")
	cityRegex, _ := regexp.Compile("^\\s*city_data\\s*:")
	cityCountryRegex, _ := regexp.Compile("^\\s*country\\s*:")

	trailers := Trailers{
		Variants:    []string{},
		Definitions: make(map[string]TrailerDefinition),
	}
	cargoes := []string{}
	companies := make(map[string]*Company)
	cities := make(map[string]City)

	for _, file := range trailerFiles {
		input, _ := ioutil.ReadFile(fmt.Sprintf("%s/%s", trailerDir, file.Name()))

		reader := bufio.NewReader(bytes.NewReader(input))

		var line string
		var err error
		for line, err = reader.ReadString('\n'); err == nil; line, err = reader.ReadString('\n') {
			if trailerRegex.MatchString(line) {
				result := strings.Split(line, ":")
				trailers.Variants = append(trailers.Variants, strings.TrimSpace(strings.Split(result[1], "\t")[0]))
			}
		}
	}
	for _, file := range trailerDefFiles {
		readTrailerDef(trailerDefDir, file, &trailers)
	}
	for _, file := range cargoFiles {
		if file.IsDir() {
			childDir := path.Join(cargoDir, file.Name())
			childFiles, _ := ioutil.ReadDir(childDir)
			for _, childFile := range childFiles {
				readTrailerDef(childDir, childFile, &trailers)
			}
			continue
		}
		readTrailerDef(cargoDir, file, &trailers)
	}
	for _, file := range cargoFiles {
		input, _ := ioutil.ReadFile(fmt.Sprintf("%s/%s", cargoDir, file.Name()))

		reader := bufio.NewReader(bytes.NewReader(input))

		var line string
		var err error
		for line, err = reader.ReadString('\n'); err == nil; line, err = reader.ReadString('\n') {
			if cargoRegex.MatchString(line) {
				result := strings.Split(line, ":")
				cargoes = append(cargoes, strings.TrimSpace(strings.Split(result[1], "\t")[0]))
			}
		}
	}
	for _, file := range companyFiles {
		if !file.IsDir() {
			continue
		}

		company, ok := companies[file.Name()]
		if !ok {
			company = &Company{
				Cities:     []string{},
				CargoesIn:  []string{},
				CargoesOut: []string{},
			}
			companies[file.Name()] = company
		}

		childDir := path.Join(companyDir, file.Name(), "editor")
		childFiles, _ := ioutil.ReadDir(childDir)
		for _, childFile := range childFiles {
			company.Cities = append(company.Cities, readCompanyCity(childDir, childFile)...)
		}

		childDir = path.Join(companyDir, file.Name(), "in")
		childFiles, _ = ioutil.ReadDir(childDir)
		for _, childFile := range childFiles {
			company.CargoesIn = append(company.CargoesIn, readCompanyCargo(childDir, childFile)...)
		}

		childDir = path.Join(companyDir, file.Name(), "out")
		childFiles, _ = ioutil.ReadDir(childDir)
		for _, childFile := range childFiles {
			company.CargoesOut = append(company.CargoesOut, readCompanyCargo(childDir, childFile)...)
		}
	}

	for _, file := range cityFiles {
		input, _ := ioutil.ReadFile(fmt.Sprintf("%s/%s", cityDir, file.Name()))

		reader := bufio.NewReader(bytes.NewReader(input))

		var line string
		var err error

		cityName := ""
		cityDef := City{}
		for line, err = reader.ReadString('\n'); err == nil; line, err = reader.ReadString('\n') {
			if cityRegex.MatchString(line) {
				if len(cityName) >0 {
					cities[cityName] = cityDef
				}

				result := strings.Split(line, ":")
				cityName = strings.TrimSpace(strings.Split(result[1], "\t")[0])
			}

			if cityCountryRegex.MatchString(line) {
				result := strings.Split(line, ":")
				cityDef.Country = strings.TrimSpace(strings.Split(result[1], "\t")[0])
			}
		}

		if len(cityName) >0 {
			cities[cityName] = cityDef
		}
	}

	output, _ := json.MarshalIndent(trailers, "", "  ")
	_ = ioutil.WriteFile(fmt.Sprintf("trailers_%s.json", *dlcName), output, 0664)
	output, _ = json.MarshalIndent(cargoes, "", "  ")
	_ = ioutil.WriteFile(fmt.Sprintf("cargoes_%s.json", *dlcName), output, 0664)
	output, _ = json.MarshalIndent(companies, "", "  ")
	_ = ioutil.WriteFile(fmt.Sprintf("companies_%s.json", *dlcName), output, 0664)
	output, _ = json.MarshalIndent(cities, "", "  ")
	_ = ioutil.WriteFile(fmt.Sprintf("cities_%s.json", *dlcName), output, 0664)
}

func readTrailerDef(dir string, file os.FileInfo, trailers *Trailers) {
	input, _ := ioutil.ReadFile(fmt.Sprintf("%s/%s", dir, file.Name()))

	reader := bufio.NewReader(bytes.NewReader(input))

	var line string
	var err error
	defName := ""
	def := TrailerDefinition{Countries: []string{}}
	for line, err = reader.ReadString('\n'); err == nil; line, err = reader.ReadString('\n') {
		if trailerDefRegex.MatchString(line) {
			if len(defName) > 0 {
				trailers.Definitions[defName] = def
			}
			result := strings.Split(line, ":")
			defName = strings.TrimSpace(strings.Split(result[1], "\t")[0])
		}
		if trailerDefCountryRegex.MatchString(line) {
			result := strings.Split(line, ":")
			def.Countries = append(def.Countries, strings.TrimSpace(strings.Split(result[1], "\t")[0]))
		}
	}

	if len(defName) > 0 {
		trailers.Definitions[defName] = def
	}
}

func readCompanyCity(dir string, file os.FileInfo) []string {
	input, _ := ioutil.ReadFile(fmt.Sprintf("%s/%s", dir, file.Name()))

	reader := bufio.NewReader(bytes.NewReader(input))

	var line string
	var err error
	results := []string{}
	for line, err = reader.ReadString('\n'); err == nil; line, err = reader.ReadString('\n') {
		if companyCityRegex.MatchString(line) {
			result := strings.Split(line, ":")
			results = append(results, strings.TrimSpace(strings.Split(result[1], "{")[0])[1:])
		}
	}

	return results
}

func readCompanyCargo(dir string, file os.FileInfo) []string {
	input, _ := ioutil.ReadFile(fmt.Sprintf("%s/%s", dir, file.Name()))

	reader := bufio.NewReader(bytes.NewReader(input))

	var line string
	var err error
	results := []string{}
	for line, err = reader.ReadString('\n'); err == nil; line, err = reader.ReadString('\n') {
		if companyCargoRegex.MatchString(line) {
			result := strings.Split(line, ":")
			results = append(results, strings.TrimSpace(strings.Split(result[1], "{")[0])[1:])
		}
	}

	return results
}
