package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"math"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func main() {
	if err := Duplicate(); err != nil {
		panic(err)
	}
}

// Duplicate merges the two waypoints files, removes duplicate waypoints and assigns unique codes
// to each based on their proximity to the km markers.
func Duplicate() error {

	markersGpx := gpx{}
	waypointsGpx := gpx{}

	robins := loadGpx("./source/robins-waypoints.gpx")
	hals := loadGpx("./source/hal-waypoints.gpx")

	markers := getMarkers()

	addWaypoints := func(wps []Waypoint) {
		for _, v := range wps {
			point := closestMarker(markers, Point{Lat: v.Lat, Lon: v.Lon})

			var found bool
			for k, p := range point.Waypoints {
				dist := distance(p.Lat, p.Lon, v.Lat, v.Lon)
				if dist < 0.01 {

					// if waypoint is within 10m of a current waypoint, don't add a new one
					name1 := normaliseName(v.Name)
					name2 := normaliseName(p.Name)
					var newName string

					// merge the name in sensible fashion:
					if name1 == name2 {
						newName = name1
					} else if strings.HasPrefix(name1, name2) {
						newName = name1
					} else if strings.HasPrefix(name2, name1) {
						newName = name2
					} else {
						newName = name1 + " / " + name2
					}
					point.Waypoints[k].Name = newName

					found = true
					break
				}
			}

			if !found {
				wp := v
				wp.Name = normaliseName(wp.Name)
				point.Waypoints = append(point.Waypoints, wp)
			}

			//fmt.Printf("D%03d%s-%02d %s\n", point.Day, point.Suffix, point.Km, normaliseName(v.Name))
		}
	}
	addWaypoints(robins.Waypoints)
	addWaypoints(hals.Waypoints)

	for _, m := range markers {

		// give the start of each day the "camp" icon:
		sym := "Camp"
		if m.Km > 0 {
			// each km marker will have a numbered icon
			sym = fmt.Sprint(m.Km)
		}
		markersGpx.Waypoints = append(markersGpx.Waypoints, Waypoint{
			Lat:  m.Lat,
			Lon:  m.Lon,
			Ele:  m.Ele,
			Name: m.String(),
			Sym:  sym,
		})

		// order the waypoints for each marker by longitude (east to west)
		sort.Slice(m.Waypoints, func(i, j int) bool {
			return m.Waypoints[i].Lon > m.Waypoints[j].Lon
		})

		for k, v := range m.Waypoints {

			var suffix = 97 + rune(k) // a, b, c...

			waypointsGpx.Waypoints = append(waypointsGpx.Waypoints, Waypoint{
				Lat:  v.Lat,
				Lon:  v.Lon,
				Ele:  v.Ele,
				Name: fmt.Sprintf("%s%s %s", m.String(), string(suffix), v.Name),
				Sym:  "red",
			})

		}
	}

	saveGpx(waypointsGpx, "waypoints.gpx")
	saveGpx(markersGpx, "markers.gpx")

	return nil

}

// Merge merges multiple gpx files into a single file.
func Merge() {

	merged := gpx{}

	dir := "./source/files"
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		panic(err)
	}
	for _, f := range files {
		if f.Name() == "merged.gpx" {
			continue
		}
		fmt.Println(f.Name())
		b, err := ioutil.ReadFile(filepath.Join(dir, f.Name()))
		if err != nil {
			panic(err)
		}
		d := xml.NewDecoder(bytes.NewBuffer(b))
		var data gpx
		if err := d.Decode(&data); err != nil {
			panic(err)
		}
		for _, rte := range data.Routes {
			rte.Name = strings.TrimSuffix(rte.Name, " on GPSies.com")
			merged.Routes = append(merged.Routes, rte)
		}
		for _, trk := range data.Tracks {
			merged.Tracks = append(merged.Tracks, trk)
		}
		for _, wpt := range data.Waypoints {
			merged.Waypoints = append(merged.Waypoints, wpt)
		}
	}
	b, err := xml.Marshal(merged)
	if err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile(filepath.Join(dir, "merged.gpx"), b, 0777); err != nil {
		panic(err)
	}
}

func distance(lat1 float64, lng1 float64, lat2 float64, lng2 float64) float64 {
	const PI float64 = 3.141592653589793

	radlat1 := float64(PI * lat1 / 180)
	radlat2 := float64(PI * lat2 / 180)

	theta := float64(lng1 - lng2)
	radtheta := float64(PI * theta / 180)

	dist := math.Sin(radlat1)*math.Sin(radlat2) + math.Cos(radlat1)*math.Cos(radlat2)*math.Cos(radtheta)

	if dist > 1 {
		dist = 1
	}

	dist = math.Acos(dist)
	dist = dist * 180 / PI
	dist = dist * 60 * 1.1515

	dist = dist * 1.609344

	return dist
}

func saveGpx(g gpx, filename string) {
	bw, err := xml.Marshal(g)
	if err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile(filename, bw, 0777); err != nil {
		panic(err)
	}
}

func loadGpx(filename string) gpx {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	var g gpx
	if err := xml.NewDecoder(bytes.NewBuffer(b)).Decode(&g); err != nil {
		panic(err)
	}
	return g
}

func closest(points []Point, p Point) int {
	minDist := -1.0
	minIndex := 0
	for k, v := range points {
		d := distance(v.Lat, v.Lon, p.Lat, p.Lon)
		if d < minDist || minDist == -1.0 {
			minDist = d
			minIndex = k
		}
	}
	return minIndex
}

func closestMarker(points []*Marker, p Point) *Marker {
	minDist := -1.0
	var minPoint *Marker
	for _, v := range points {
		d := distance(v.Lat, v.Lon, p.Lat, p.Lon)
		if d < minDist || minDist == -1.0 {
			minDist = d
			minPoint = v
		}
	}
	return minPoint
}

func whichDay(routes gpx, dayStartPoints []Point, p Point) string {

	i := closest(dayStartPoints, Point{Lat: p.Lat, Lon: p.Lon})

	if i == 0 {
		return routes.Routes[i].Name
	}

	// if not the first or last day, work out if it's nearer d+1 or d-1

	prevDay := routes.Routes[i-1].Points[0]

	var nextDay Point
	// for the last day, we use the last route point in the last day
	if i == len(routes.Routes)-1 {
		nextDay = routes.Routes[i].Points[len(routes.Routes[i].Points)-1]
	} else {
		nextDay = routes.Routes[i+1].Points[0]
	}

	distPlusOne := distance(p.Lat, nextDay.Lat, p.Lon, nextDay.Lon)
	distMinusOne := distance(p.Lat, prevDay.Lat, p.Lon, prevDay.Lon)

	if distPlusOne < distMinusOne {
		// closer to d+1 => on day i
		return routes.Routes[i].Name
	} else {
		// closer to d-1 => on day i-1
		return routes.Routes[i-1].Name
	}
}

type Marker struct {
	Point
	Day       int
	Suffix    string
	Km        int
	Waypoints []Waypoint
}

func (tp *Marker) String() string {
	return fmt.Sprintf("D%03d%s-%02d", tp.Day, tp.Suffix, tp.Km)
}

func getMarkers() []*Marker {
	var markers []*Marker
	data := loadGpx("./source/routes.gpx")
	for _, rte := range data.Routes {
		var dist float64

		nextD := 0
		nextF := 0.0

		day, suffix := getDayNum(rte.Name)

		for i, v := range rte.Points {
			if i > 0 {
				prev := rte.Points[i-1]
				dist += distance(prev.Lat, prev.Lon, v.Lat, v.Lon)
				if dist > nextF {
					markers = append(markers, &Marker{
						Point:  v,
						Day:    day,
						Suffix: suffix,
						Km:     nextD,
					})
					nextF += 1.0
					nextD += 1
				}
			}
		}
	}
	return markers
}

func getDayNum(name string) (int, string) {
	var suffix string
	name = strings.TrimPrefix(name, "D")
	if strings.HasSuffix(name, "N") {
		suffix = "N"
		name = strings.TrimSuffix(name, "N")
	}
	if strings.HasSuffix(name, "S") {
		suffix = "S"
		name = strings.TrimSuffix(name, "S")
	}
	num, err := strconv.Atoi(name)
	if err != nil {
		panic(err)
	}
	return num, suffix
}

var robinNameRegex = regexp.MustCompile(`[\d ]{4}(.*)`)
var halSuffixRegex = regexp.MustCompile(`(.*)-[A-Z]{2}\d?`)
var halPrefixRegex = regexp.MustCompile(`S\d-(.*)`)

func normaliseName(name string) string {
	if robinNameRegex.MatchString(name) {
		name = robinNameRegex.ReplaceAllString(name, "$1")
	}
	if halSuffixRegex.MatchString(name) {
		name = halSuffixRegex.ReplaceAllString(name, "$1")
	}
	if halPrefixRegex.MatchString(name) {
		name = halPrefixRegex.ReplaceAllString(name, "$1")
	}
	var newName string
	var prevChar string
	for k, v := range name {
		s := string(v)
		if s == "-" || s == "_" {
			newName += " "
			prevChar = " "
			continue
		}
		if k != 0 && strings.Contains("QWERTYUIOPASDFGHJKLZXCVBNM1234567890", s) && !strings.Contains(" QWERTYUIOPASDFGHJKLZXCVBNM1234567890", prevChar) {
			// upper case or number => insert space
			newName += " "
		}
		newName += s
		prevChar = s
	}
	return newName
}

type gpx struct {
	Waypoints []Waypoint `xml:"wpt"`
	Tracks    []Track    `xml:"trk"`
	Routes    []Route    `xml:"rte"`
}

type Waypoint struct {
	Lat  float64 `xml:"lat,attr"`
	Lon  float64 `xml:"lon,attr"`
	Ele  float64 `xml:"ele"`
	Name string  `xml:"name"`
	Sym  string  `xml:"sym"`
}

type Route struct {
	Name   string  `xml:"name"`
	Points []Point `xml:"rtept"`
}

type Point struct {
	Lat float64 `xml:"lat,attr"`
	Lon float64 `xml:"lon,attr"`
	Ele float64 `xml:"ele"`
}

type Track struct {
	Name     string         `xml:"name"`
	Segments []TrackSegment `xml:"trkseg"`
}

type TrackSegment struct {
	Points []Point `xml:"trkpt"`
}
