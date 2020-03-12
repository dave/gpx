package main

import (
	"bytes"
	"fmt"
	"image/color"
	"image/jpeg"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	sm "github.com/flopp/go-staticmaps"
	"github.com/golang/geo/s2"
	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
)

var ApiKey string

func DrawElevations() error {
	dir := "/Users/dave/Dropbox/GHT/GPX files corrected with waypoints"
	out := "/Users/dave/Downloads/GPX files elevations"

	/*
		type wpdata struct {
			Waypoint
			dist float64 // distance along the route at minimum proximity
			prox float64 // proximity to the route
		}
	*/

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".gpx") {
			continue
		}
		leg, err := strconv.Atoi(file.Name()[1:4])
		if leg != 1 {
			//continue
		}
		if err != nil {
			return err
		}
		g := loadGpx(filepath.Join(dir, file.Name()))
		fmt.Println(leg, len(g.Routes[0].Points))

		/*
			var waypoints []*wpdata
			for _, wp := range g.Waypoints {
				waypoints = append(waypoints, &wpdata{Waypoint: wp, prox: -1})
			}
		*/

		series := chart.ContinuousSeries{}
		var d, minEle, maxEle float64
		for i, point := range g.Routes[0].Points {
			if i > 0 {
				last := g.Routes[0].Points[i-1]
				d += distance(last.Lat, last.Lon, point.Lat, point.Lon) * 1000
			}
			series.XValues = append(series.XValues, d)
			series.YValues = append(series.YValues, point.Ele)
			if point.Ele > maxEle || maxEle == 0 {
				maxEle = point.Ele
			}
			if point.Ele < minEle || minEle == 0 {
				minEle = point.Ele
			}
			/*
				// calculate proximity of all waypoints
				for _, w := range waypoints {
					prox := distance(w.Lat, w.Lon, point.Lat, point.Lon) * 1000
					if prox < w.prox || w.prox == -1 {
						w.prox = prox
						w.dist = d
					}
				}
			*/
		}

		/*
			annotations := chart.AnnotationSeries{}
			for _, w := range waypoints {
				if w.prox > 200 {
					continue
				}
				name := fmt.Sprintf("%s %dm", w.Name, w.Ele)
				annotations.Annotations = append(annotations.Annotations, chart.Value2{XValue: w.dist, YValue: w.Ele, Label: name})
			}
		*/

		maxX := math.Ceil(d/1000) * 1000
		minY := math.Floor(minEle/1000) * 1000
		maxY := math.Ceil(maxEle/1000) * 1000

		if maxY-maxEle < 100 {
			maxY += 1000
		}

		if minEle-minY < 100 && minY > 0 {
			minY -= 1000
		}

		plot := chart.Chart{}
		plot.Series = []chart.Series{series}

		plot.YAxis.Name = "Elevation"
		plot.YAxis.Range = &chart.ContinuousRange{Min: minY, Max: maxY}
		plot.YAxis.Style.Show = true
		for i := minY; i <= maxY; i += 1000.0 {
			plot.YAxis.GridLines = append(plot.YAxis.GridLines, chart.GridLine{Value: i})
			plot.YAxis.Ticks = append(plot.YAxis.Ticks, chart.Tick{Value: i, Label: fmt.Sprintf("%dm", int(i))})
			for j := i + 100; j <= i+900; j += 100 {
				if j == i+500 {
					plot.YAxis.GridLines = append(plot.YAxis.GridLines, chart.GridLine{Value: j, IsMinor: true})
				} else {
					plot.YAxis.GridLines = append(plot.YAxis.GridLines, chart.GridLine{Value: j, IsMinor: true, Style: chart.Style{
						Show:            true,
						StrokeWidth:     1,
						StrokeColor:     drawing.Color{R: 0xDD, G: 0xDD, B: 0xDD, A: 0xFF},
						StrokeDashArray: []float64{5.0, 5.0},
					}})
				}
			}
		}
		plot.YAxis.GridMinorStyle = chart.Style{
			Show:        true,
			StrokeWidth: 1,
			StrokeColor: drawing.Color{R: 0xDD, G: 0xDD, B: 0xDD, A: 0xFF},
		}
		plot.YAxis.GridMajorStyle = chart.Style{
			Show:        true,
			StrokeWidth: 1,
			StrokeColor: drawing.Color{R: 0x66, G: 0x66, B: 0x66, A: 0xFF},
		}

		plot.XAxis.Name = "Distance"
		plot.XAxis.Range = &chart.ContinuousRange{Min: 0, Max: maxX}
		plot.XAxis.Style.Show = true
		for i := 0.0; i <= maxX; i += 1000.0 {
			plot.XAxis.GridLines = append(plot.XAxis.GridLines, chart.GridLine{Value: i})
			plot.XAxis.Ticks = append(plot.XAxis.Ticks, chart.Tick{Value: i, Label: fmt.Sprintf("%dkm", int(i)/1000)})
		}
		plot.XAxis.GridMajorStyle = chart.Style{
			Show:        true,
			StrokeWidth: 1,
			StrokeColor: chart.ColorAlternateLightGray,
		}

		plot.Height = int((1500/maxX)*(maxY-minY)) + 35
		plot.Width = 1500

		writeChart(plot, filepath.Join(out, fmt.Sprintf("E%03d.png", leg)))
	}

	return nil
}

func writeChart(c chart.Chart, fpath string) {
	f, err := os.Create(fpath)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := c.Render(chart.PNG, f); err != nil {
		panic(err)
	}
}

// Draw draws maps of each day of the route using OpenStreetMap and the GPX routes
func Draw() error {
	dir := "/Users/dave/Dropbox/GHT/GPX files corrected with waypoints"
	out := "/Users/dave/Downloads/GPX files maps"
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	minZoom := 0
	type data struct {
		Leg int
		Gpx gpx
	}
	var routes []data
	var legs []int
	routesM := map[int]gpx{}
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".gpx") {
			continue
		}
		leg, err := strconv.Atoi(file.Name()[1:4])
		if err != nil {
			return err
		}
		g := loadGpx(filepath.Join(dir, file.Name()))
		routesM[leg] = g
		legs = append(legs, leg)
	}
	sort.Ints(legs)
	for _, leg := range legs {
		routes = append(routes, data{
			Leg: leg,
			Gpx: routesM[leg],
		})
	}

	for i, dat := range routes {
		if dat.Leg != 35 {
			//continue
		}

		rte := dat.Gpx.Routes[0]
		waypoints := dat.Gpx.Waypoints

		ctx := sm.NewContext()

		/*

			Create a file apikey.go, with the contents:

			package main

			func init() {
				ApiKey = "YOUR_API_KEY"
			}

		*/

		t := new(sm.TileProvider)
		t.Name = "thunderforest-landscape"
		t.Attribution = "Maps (c) Thundeforest; Data (c) OSM and contributors, ODbL"
		t.TileSize = 256
		t.URLPattern = "https://tile.thunderforest.com/landscape/%[2]d/%[3]d/%[4]d.png?apikey=" + ApiKey

		ctx.SetTileProvider(t)

		ctx.SetSize(1200, 1200)

		var minLat, maxLat, minLon, maxLon float64
		for i, point := range rte.Points {
			if i == 0 || point.Lat < minLat {
				minLat = point.Lat
			}
			if i == 0 || point.Lat > maxLat {
				maxLat = point.Lat
			}
			if i == 0 || point.Lon < minLon {
				minLon = point.Lon
			}
			if i == 0 || point.Lon > maxLon {
				maxLon = point.Lon
			}
		}

		bb := s2.NewRectBounder()
		bb.AddPoint(s2.PointFromLatLng(s2.LatLngFromDegrees(minLat, minLon)))
		bb.AddPoint(s2.PointFromLatLng(s2.LatLngFromDegrees(maxLat, maxLon)))
		//ctx.SetBoundingBox(bb.RectBound())

		center := bb.RectBound().Center()
		if dat.Leg == 87 {
			// set centre a bit to the west for leg 87 (so Chap Chu is visible)
			center.Lng = center.Lng - 0.0007
		}
		ctx.SetCenter(center)
		if dat.Leg == 102 {
			ctx.SetZoom(12)
		} else {
			ctx.SetZoom(13)
		}
		//ctx.SetZoom(14)
		// Best: 12

		{

			otherLegColor := color.RGBA{0, 0, 0x44, 0x44}
			thisLegColor := color.RGBA{0xcc, 0, 0, 0xcc}
			drawPath := func(points []Point, c color.RGBA) {
				var d float64
				for i, v := range points {
					if i > 0 {
						d += distance(v.Lat, v.Lon, points[i-1].Lat, points[i-1].Lon)
						if d > 0.2 {
							d = 0
						} else {
							continue
						}
					}
					ctx.AddCircle(sm.NewCircle(
						s2.LatLngFromDegrees(v.Lat, v.Lon),
						color.RGBA{0xff, 0, 0, 0xff},
						c,
						50.0,
						0.0,
					))
				}

				/*var segments []s2.LatLng
				for _, v := range points {
					segments = append(segments, s2.LatLngFromDegrees(v.Lat, v.Lon))
				}
				ctx.AddPath(sm.NewPath(segments, color.RGBA{0, 0, 0xff, alpha}, 1.0))*/
			}
			if i > 0 {
				drawPath(routes[i-1].Gpx.Routes[0].Points, otherLegColor)
			}
			if i > 1 {
				drawPath(routes[i-2].Gpx.Routes[0].Points, otherLegColor)
			}
			if i < len(routes)-1 {
				drawPath(routes[i+1].Gpx.Routes[0].Points, otherLegColor)
			}
			if i < len(routes)-2 {
				drawPath(routes[i+2].Gpx.Routes[0].Points, otherLegColor)
			}

			drawPath(rte.Points, thisLegColor)

			/*var segments []s2.LatLng
			for _, v := range rte.Points {
				segments = append(segments, s2.LatLngFromDegrees(v.Lat, v.Lon))
			}
			ctx.AddPath(sm.NewPath(segments, color.RGBA{0xff, 0, 0, 0xff}, 1.0))*/

		}

		ctx.AddMarker(
			sm.NewMarker(
				s2.LatLngFromDegrees(rte.Points[0].Lat, rte.Points[0].Lon),
				color.RGBA{0, 0xcc, 0, 0xff},
				15.0,
			),
		)
		/*ctx.AddMarker(
			sm.NewMarker(
				s2.LatLngFromDegrees(rte.Points[len(rte.Points)-1].Lat, rte.Points[len(rte.Points)-1].Lon),
				color.RGBA{0xcc, 0, 0, 0xff},
				15.0,
			),
		)*/
		for _, p := range waypoints {
			m := sm.NewMarker(
				s2.LatLngFromDegrees(p.Lat, p.Lon),
				color.RGBA{0xff, 0, 0, 0xff},
				15.0,
			)
			m.Label = p.Name
			m.LabelColor = color.Black
			ctx.AddMarker(m)
		}

		img, zoom, err := ctx.Render()
		if err != nil {
			return err
		}
		if minZoom == 0 || zoom < minZoom {
			minZoom = zoom
		}
		fmt.Println(dat.Leg, zoom)

		buf := &bytes.Buffer{}
		if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: 90}); err != nil {
			return err
		}

		if err := ioutil.WriteFile(filepath.Join(out, fmt.Sprintf("L%03d.jpg", dat.Leg)), buf.Bytes(), 0777); err != nil {
			return err
		}
	}
	fmt.Println(minZoom)
	return nil
}
