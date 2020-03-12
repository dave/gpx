package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"math"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

func main() {
	//if err := ProcessFinalRoutesTweakWaypoints(); err != nil {
	//if err := CalcStats(); err != nil {
	if err := ProcessFinalRoutesAll(); err != nil {
		//if err := Draw(); err != nil {
		//if err := DrawElevations(); err != nil {
		panic(err)
	}
}

func ProcessFinalRoutesTweakWaypoints() error {
	inDir := `/Users/dave/Dropbox/GHT/GPX files corrected with waypoints`
	outDir := `/Users/dave/Dropbox/GHT/GPX files corrected with waypoints tracks`

	routeFiles, err := ioutil.ReadDir(inDir)
	if err != nil {
		return err
	}

	for _, fileInfo := range routeFiles {
		if !strings.HasSuffix(fileInfo.Name(), ".gpx") {
			continue
		}
		g := loadGpx(filepath.Join(inDir, fileInfo.Name()))
		legNumber, err := strconv.Atoi(fileInfo.Name()[1:4])
		if err != nil {
			return err
		}
		for i := range g.Waypoints {
			g.Waypoints[i].Sym = ""
		}
		var points []TrackPoint
		for _, point := range g.Routes[0].Points {
			points = append(points, TrackPoint{Point: point})
		}
		t := Track{
			Name:     g.Routes[0].Name,
			Desc:     g.Routes[0].Desc,
			Segments: []TrackSegment{{Points: points}},
		}
		g.Routes = []Route{}
		g.Tracks = []Track{t}
		saveGpx(g, filepath.Join(outDir, fmt.Sprintf("L%03d.gpx", legNumber)))
	}
	return nil
}

func ProcessFinalRoutesAll() error {
	if err := ProcessFinalRoutes(true); err != nil {
		return err
	}
	if err := ProcessFinalRoutes(false); err != nil {
		return err
	}
	return nil
}

func ProcessFinalRoutes(mapsme bool) error {

	b, err := ioutil.ReadFile(`/Users/dave/src/youtube/trailnotes.json`)
	if err != nil {
		return err
	}
	var notes TrailNotesSheetStruct
	if err := json.Unmarshal(b, &notes); err != nil {
		return err
	}

	legsByLeg := map[int]*LegStruct{}

	legs := notes.Legs
	for i, leg := range legs {
		if i == 0 {
			leg.From = "Taplejung"
		} else {
			leg.From = legs[i-1].To
		}
		for _, waypoint := range notes.Waypoints {
			if waypoint.Leg == leg.Leg {
				leg.Waypoints = append(leg.Waypoints, waypoint)
			}
		}
		for _, pass := range notes.Passes {
			if pass.Leg == leg.Leg {
				leg.Passes = append(leg.Passes, pass)
			}
		}
		if leg.Vlog != nil {
			days := strings.Split(fmt.Sprint(leg.Vlog), ",")
			for _, day := range days {
				d, err := strconv.Atoi(day)
				if err != nil {
					return err
				}
				leg.Days = append(leg.Days, d)
			}
		}
		legsByLeg[leg.Leg] = leg
	}

	inDir := `/Users/dave/Dropbox/GHT/GPX files corrected with waypoints`
	outDir := `/Users/dave/Dropbox/GHT/GPX files corrected final`

	routeFiles, err := ioutil.ReadDir(inDir)
	if err != nil {
		return err
	}

	var out gpx

	for _, fileInfo := range routeFiles {
		if !strings.HasSuffix(fileInfo.Name(), ".gpx") {
			continue
		}
		g := loadGpx(filepath.Join(inDir, fileInfo.Name()))
		legNumber, err := strconv.Atoi(fileInfo.Name()[1:4])
		if err != nil {
			return err
		}
		//fmt.Println(legNumber, len(g.Routes[0].Points), len(g.Waypoints))

		leg := legsByLeg[legNumber]

		// check all waypoints
		for _, waypointFromNotes := range leg.Waypoints {
			var found bool
			for _, waypointFromGpx := range g.Waypoints {
				if fmt.Sprintf("L%03d %s", legNumber, waypointFromNotes.Name) == waypointFromGpx.Name {
					found = true
					waypointFromNotes.Lat = waypointFromGpx.Lat
					waypointFromNotes.Lon = waypointFromGpx.Lon
					waypointFromNotes.Ele = waypointFromGpx.Ele
					//fmt.Printf("%d\t%s\t%f\n", legNumber, waypointFromNotes.Name, waypointFromGpx.Ele)
					break
				}
			}
			if !found {
				return fmt.Errorf("missing waypoint %s", waypointFromNotes.Name)
			}
		}

		for _, waypointFromGpx := range g.Waypoints {
			var found bool
			for _, waypointFromNotes := range leg.Waypoints {
				if fmt.Sprintf("L%03d %s", legNumber, waypointFromNotes.Name) == waypointFromGpx.Name {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("missing waypoint %s", waypointFromGpx.Name)
			}
		}

		for _, pass := range leg.Passes {
			var found bool
			for _, waypointFromGpx := range g.Waypoints {
				if fmt.Sprintf("L%03d %s", legNumber, pass.Pass) == waypointFromGpx.Name {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("missing pass %s", pass.Pass)
			}
		}
		out.Routes = append(out.Routes, Route{
			Name:   fmt.Sprintf("L%03d %s to %s", leg.Leg, leg.From, leg.To),
			Desc:   fmt.Sprintf("%s", leg.Notes),
			Points: g.Routes[0].Points,
		})
		if mapsme {
			// maps.me doesn't show descriptions for routes so we add a dummy waypoint
			out.Waypoints = append(out.Waypoints, Waypoint{
				Lat:  g.Routes[0].Points[0].Lat,
				Lon:  g.Routes[0].Points[0].Lon,
				Ele:  g.Routes[0].Points[0].Ele,
				Name: fmt.Sprintf("L%03d %s to %s", leg.Leg, leg.From, leg.To),
				Desc: leg.Notes,
			})
		}
		for _, w := range leg.Waypoints {
			out.Waypoints = append(out.Waypoints, Waypoint{
				Lat:  w.Lat,
				Lon:  w.Lon,
				Ele:  w.Ele,
				Name: fmt.Sprintf("L%03d %s", leg.Leg, w.Name),
				Desc: w.Notes,
			})
		}

	}

	if mapsme {
		saveGpx(out, filepath.Join(outDir, "routes-for-maps-me-v3.gpx"))
	} else {
		saveGpx(out, filepath.Join(outDir, "routes-v3.gpx"))
	}

	return nil

}

func CalcStats() error {

	routesDir := "/Users/dave/Dropbox/GHT/GPX files corrected with waypoints"
	routeFiles, err := ioutil.ReadDir(routesDir)
	if err != nil {
		return err
	}

	for _, fileInfo := range routeFiles {

		if !strings.HasSuffix(fileInfo.Name(), ".gpx") {
			continue
		}

		leg, err := strconv.Atoi(fileInfo.Name()[1:4])
		if err != nil {
			return err
		}

		if leg != 72 {
			//continue
		}

		g := loadGpx(filepath.Join(routesDir, fileInfo.Name()))
		if len(g.Routes) != 1 {
			return fmt.Errorf("not 1 route for %q", fileInfo.Name())
		}

		var length, climb, descent, start, end, top, bottom float64
		for i, current := range g.Routes[0].Points {
			if i == 0 {
				start = current.Ele
				top = current.Ele
				bottom = current.Ele
			}
			if i == len(g.Routes[0].Points)-1 {
				end = current.Ele
			}
			if i == 0 {
				continue
			}

			// work out distance delta
			previous := g.Routes[0].Points[i-1]
			horizontal := distance(current.Lat, current.Lon, previous.Lat, previous.Lon)

			// work out elevation delta
			var vertical float64
			climbing := current.Ele > previous.Ele
			if climbing {
				vertical = current.Ele - previous.Ele
			} else {
				vertical = previous.Ele - current.Ele
			}

			if vertical > 50 {
				// discard outlier points
				continue
			}

			verticalkm := vertical / 1000.0

			total := math.Sqrt(horizontal*horizontal + verticalkm*verticalkm)

			if current.Ele > top {
				top = current.Ele
			}
			if current.Ele < bottom {
				bottom = current.Ele
			}

			length += total

			if climbing {
				climb += vertical
			} else {
				descent += vertical
			}
		}
		//fmt.Printf("%d\t%f\t%f\t%f\t%f\t%f\t%f\t%f\n", leg, length, climb, descent, start, end, top, bottom)
		fmt.Printf("%f\t%f\t%f\t%f\t%f\t%f\t%f\n", length, climb, descent, start, end, top, bottom)
		//start = end
		//end = start
		//if climb > 3000 {
		//fmt.Printf("%d\t%f\t%f\n", leg, length, climb)
		//}
	}

	return nil
}

func SplitInReach() error {
	dir := "/Users/dave/Dropbox/GHT/GPS tracks/InReach tracks split"
	g := loadGpx("/Users/dave/Dropbox/GHT/GPS tracks/InReach tracks and waypoints original.gpx")
	for _, track := range g.Tracks {
		for i, segment := range track.Segments {
			if len(segment.Points) < 10 {
				continue
			}
			name := track.Name
			if len(track.Segments) > 1 {
				name += fmt.Sprintf(" (#%d)", i)
			}
			gOut := gpx{
				Tracks: []Track{{
					Segments: []TrackSegment{
						segment,
					},
				}},
			}
			saveGpx(gOut, filepath.Join(dir, fmt.Sprintf("%s.gpx", name)))
		}
	}
	return nil
}

func MergeTracks() error {
	dataBytes, err := ioutil.ReadFile("/Users/dave/src/gpx/data.json")
	if err != nil {
		return err
	}
	var data []DataStruct
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		return err
	}

	var days []*DayInfo
	legs := map[int]*LegData{}
	for _, item := range data {

		if item.Day != 0 {
			correctedDate := item.Date.Add(time.Hour) // not sure why this is!!!
			days = append(days, &DayInfo{Day: item.Day, Date: correctedDate})
		}

		if item.Leg != 0 {
			if legs[item.Leg] == nil {
				legs[item.Leg] = &LegData{
					Leg:  item.Leg,
					From: item.LegFrom,
					To:   item.LegTo,
				}
			}
			if item.Day != 0 {
				correctedDate := item.Date.Add(time.Hour) // not sure why this is!!!
				legs[item.Leg].RelevantDays = append(legs[item.Leg].RelevantDays, DayInfo{Day: item.Day, Date: correctedDate})
			}
		}
	}

	var ordered []int
	for leg := range legs {
		ordered = append(ordered, leg)
	}
	sort.Ints(ordered)

	/*
		for _, leg := range ordered {
			if legs[leg] != nil {
				fmt.Println(leg, legs[leg].RelevantDays)
			}
		}
	*/

	routesDir := "/Users/dave/Dropbox/GHT/Final GPX files/Daily routes individual files"
	routeFiles, err := ioutil.ReadDir(routesDir)
	if err != nil {
		return err
	}

	for _, fileInfo := range routeFiles {
		g := loadGpx(filepath.Join(routesDir, fileInfo.Name()))
		if g.Routes[0].Name == "D088N" || g.Routes[0].Name == "D089N" {
			continue
		}
		leg, err := strconv.Atoi(g.Routes[0].Name[1:4])
		if err != nil {
			return err
		}
		if legs[leg] == nil {
			continue
		}
		legs[leg].Gpx = g
	}
	/*
		for _, l := range legs {
			fmt.Println(l.Gpx.Routes[0].Name, len(l.Gpx.Routes[0].Points))
		}
	*/

	inReachTracksFilename := "/Users/dave/Dropbox/GHT/GPS tracks/InReach tracks and waypoints original.gpx"
	myTracksTracksFilename := "/Users/dave/Dropbox/GHT/GPS tracks/myTracks merged.gpx"
	viewRangerTracksFilename := "/Users/dave/Dropbox/GHT/GPS tracks/ViewRanger/ViewRanger only my tracks.gpx"

	loc, err := time.LoadLocation("Asia/Kathmandu")
	if err != nil {
		return err
	}

	toDate := func(t time.Time) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
	}

	matchDate := func(t1, t2 time.Time) bool {
		return t1.Year() == t2.Year() && t1.Month() == t2.Month() && t1.Day() == t2.Day()
	}

	readAndMergeTracks := func(filename, source string) error {
		g := loadGpx(filename)
		for _, track := range g.Tracks {

			if track.Name == "Dave and Mathilde (0)" {
				continue
			}

			//2019-04-18 08:06:57
			var nameHasDate bool
			nameDate, err := time.Parse("2006-01-02 15:04:05", track.Name)
			if err == nil {
				nameHasDate = true
				nameDate = toDate(nameDate)
			}

			for segmenti, segment := range track.Segments {
				if len(segment.Points) < 10 {
					continue
				}
				trackName := fmt.Sprintf("%s (%s)", track.Name, source)
				if len(track.Segments) > 1 {
					trackName += fmt.Sprintf(" [#%d]", segmenti+1)
				}
				first := toDate(segment.Points[0].Time.In(loc))
				last := toDate(segment.Points[len(segment.Points)-1].Time.In(loc))

				for _, day := range days {
					var found bool
					for current := first; !current.After(last); current = current.Add(time.Hour * 24) {
						if matchDate(current, day.Date) {
							//fmt.Printf("matched date %v with %v\n", current, dayInfo.Date)
							found = true
							break
						}
					}
					if nameHasDate && matchDate(nameDate, day.Date) {
						found = true
					}
					if found {
						day.Gpx.Tracks = append(day.Gpx.Tracks, Track{Name: trackName, Segments: []TrackSegment{segment}})
					}
				}

				for _, leg := range legs {
					for _, dayInfo := range leg.RelevantDays {
						var found bool
						for current := first; !current.After(last); current = current.Add(time.Hour * 24) {
							if matchDate(current, dayInfo.Date) {
								//fmt.Printf("matched date %v with %v\n", current, dayInfo.Date)
								found = true
								break
							}
						}
						if nameHasDate && matchDate(nameDate, dayInfo.Date) {
							fmt.Printf("matched date %v with %v\n", nameDate, dayInfo.Date)
							found = true
						}
						if found {
							leg.Tracks = append(leg.Tracks, Track{Name: trackName, Segments: []TrackSegment{segment}})
						}
					}
				}
			}
		}
		return nil
	}

	if err := readAndMergeTracks(inReachTracksFilename, "InReach"); err != nil {
		return err
	}
	if err := readAndMergeTracks(myTracksTracksFilename, "MyTracks"); err != nil {
		return err
	}
	if err := readAndMergeTracks(viewRangerTracksFilename, "ViewRanger"); err != nil {
		return err
	}

	/*
		for _, leg := range legs {
			fmt.Printf("Leg %d, %s to %s. %d tracks:\n", leg.Leg, leg.From, leg.To, len(leg.Tracks))
			for _, track := range leg.Tracks {
				fmt.Printf("%s: %d points\n", track.Name, len(track.Segments[0].Points))
			}
			fmt.Println("")
		}
	*/

	/*
		extraTracks := map[int][]Track{}
		// add previous and next days tracks
		for i, legId := range ordered {
			if i > 0 {
				for _, track := range legs[ordered[i-1]].Tracks {
					extraTracks[legId] = append(extraTracks[legId], track)
				}
				//fmt.Printf("Leg %d: Found %d tracks in previous leg (%d)\n", legId, len(legs[ordered[i-1]].Tracks), ordered[i-1])
			}
			if i < len(legs)-1 {
				for _, track := range legs[ordered[i+1]].Tracks {
					extraTracks[legId] = append(extraTracks[legId], track)
				}
				//fmt.Printf("Leg %d: Found %d tracks in next leg (%d)\n", legId, len(legs[ordered[i+1]].Tracks), ordered[i+1])
			}
		}
		for i, tracks := range extraTracks {
			for _, track := range tracks {
				legs[i].Tracks = append(legs[i].Tracks, track)
			}
		}
	*/

	/*
		for _, leg := range legs {
			fmt.Printf("Leg %d, %s to %s. %d tracks:\n", leg.Leg, leg.From, leg.To, len(leg.Tracks))
			for _, track := range leg.Tracks {
				fmt.Printf("%s: %d points\n", track.Name, len(track.Segments[0].Points))
			}
			fmt.Println("")
		}
	*/

	outDir := "/Users/dave/Dropbox/GHT/GPX tracks by day"
	for _, day := range days {
		saveGpx(day.Gpx, filepath.Join(outDir, fmt.Sprintf("D%03d.gpx", day.Day)))
	}

	/*
		outDir := "/Users/dave/Dropbox/GHT/GPX files corrections merged"

		for _, leg := range legs {
			g := gpx{
				//Routes: []Route{
				//	{
				//		Name:   fmt.Sprintf("Leg %d: %s to %s", leg.Leg, leg.From, leg.To),
				//		Points: leg.Gpx.Routes[0].Points,
				//	},
				//},
				Tracks: leg.Tracks,
			}
			saveGpx(g, filepath.Join(outDir, fmt.Sprintf("L%03d.gpx", leg.Leg)))
		}
	*/

	/*for _, leg := range legs {
		dir := filepath.Join(outDir, fmt.Sprintf("L%03d", leg.Leg))
		if err := os.MkdirAll(dir, 0777); err != nil {
			return err
		}
		routeGpx := gpx{
			Routes: []Route{
				{
					Name:   fmt.Sprintf("Leg %d: %s to %s", leg.Leg, leg.From, leg.To),
					Points: leg.Gpx.Routes[0].Points,
				},
			},
		}
		saveGpx(routeGpx, filepath.Join(dir, fmt.Sprintf("L%03d.gpx", leg.Leg)))
		for _, track := range leg.Tracks {
			g := gpx{
				Tracks: []Track{track},
			}
			saveGpx(g, filepath.Join(dir, fmt.Sprintf("%s.gpx", track.Name)))
		}
	}*/

	return nil
}

type LegData struct {
	Leg          int
	From, To     string
	RelevantDays []DayInfo
	Gpx          gpx
	Tracks       []Track
}

type DayInfo struct {
	Day  int
	Date time.Time
	Gpx  gpx
}

type DataStruct struct {
	Day            int
	Date           time.Time
	DayFrom, DayTo string
	Leg            int
	LegFrom, LegTo string
	Notes          string
}

func Convert() error {
	dir1 := "/Users/dave/Dropbox/GHT/old/Daily routes individual files"
	dir2 := "/Users/dave/Dropbox/GHT/old/Daily routes individual files, converted to tracks"
	files, err := ioutil.ReadDir(dir1)
	if err != nil {
		panic(err)
	}
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".gpx") {
			continue
		}
		g1 := loadGpx(filepath.Join(dir1, f.Name()))
		g2 := gpx{}

		for _, v := range g1.Routes {
			t := Track{}
			t.Name = v.Name
			s := TrackSegment{}
			for _, p := range v.Points {
				s.Points = append(s.Points, TrackPoint{Point: p})
			}
			t.Segments = append(t.Segments, s)
			g2.Tracks = append(g2.Tracks, t)
		}
		saveGpx(g2, filepath.Join(dir2, f.Name()))
	}
	return nil
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

		if m.Km%3 == 0 {
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
		}

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
	saveGpx(markersGpx, "markers-every-3-km.gpx")

	return nil

}

// Merge merges multiple gpx files into a single file.
func Merge() error {

	merged := gpx{}

	dir := "/Users/dave/Dropbox/GHT/GPS tracks/myTracks"
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
	return nil
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
		panic(fmt.Errorf("error reading file %q: %w", filename, err))
	}
	var g gpx
	if err := xml.NewDecoder(bytes.NewBuffer(b)).Decode(&g); err != nil {
		panic(fmt.Errorf("error decoding xml for %q: %w", filename, err))
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
	Desc string  `xml:"desc"`
}

type Route struct {
	Name   string  `xml:"name"`
	Desc   string  `xml:"desc"`
	Points []Point `xml:"rtept"`
}

type Point struct {
	Lat float64 `xml:"lat,attr"`
	Lon float64 `xml:"lon,attr"`
	Ele float64 `xml:"ele"`
}

type TrackPoint struct {
	Point
	Time *time.Time `xml:"time,omitempty"`
}

type Track struct {
	Name     string         `xml:"name"`
	Desc     string         `xml:"desc"`
	Segments []TrackSegment `xml:"trkseg"`
}

type TrackSegment struct {
	Points []TrackPoint `xml:"trkpt"`
}

type TrailNotesSheetStruct struct {
	Legs      []*LegStruct
	Waypoints []*WaypointStruct
	Passes    []*PassStruct
}

type LegStruct struct {
	Leg  int
	Vlog interface{}

	To                                              string
	Length, Climb, Descent, Start, End, Top, Bottom float64
	Route, Trail, Quality                           int
	Lodge                                           string
	Notes                                           string

	From      string
	Waypoints []*WaypointStruct
	Passes    []*PassStruct
	Days      []int

	RouteString, TrailString, LodgeString, QualityString string
}

type WaypointStruct struct {
	Leg           int
	Name, Notes   string
	Lat, Lon, Ele float64
}

type PassStruct struct {
	Leg    int
	Pass   string
	Height float64
}
