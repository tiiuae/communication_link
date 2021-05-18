package worldengine

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type PreplannedPoint struct {
	X float64 `json:"lat"`
	Y float64 `json:"lon"`
	Z float64 `json:"alt"`
}

func loadPrelanned(droneName string) ([]PreplannedPoint, error) {
	filename := fmt.Sprintf("./flightpath-%s.json", droneName)
	points, err := loadPrelannedFile(filename)
	if err != nil {
		return loadPrelannedFile("./flightpath.json")
	}

	return points, nil
}

func loadPrelannedFile(filename string) ([]PreplannedPoint, error) {
	text, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var path []PreplannedPoint
	err = json.Unmarshal(text, &path)
	if err != nil {
		return nil, err
	}

	return path, nil
}
