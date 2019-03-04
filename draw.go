package main

import (
	"bytes"
	"fmt"
	"image/color"
	"image/jpeg"
	"io/ioutil"

	"github.com/golang/geo/s2"

	"github.com/flopp/go-staticmaps"
)

var ApiKey string

// Draw draws maps of each day of the route using OpenStreetMap and the GPX routes
func Draw() error {
	data := loadGpx("./source/routes.gpx")
	for _, rte := range data.Routes {
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

		ctx.SetSize(800, 800)

		ctx.AddMarker(
			sm.NewMarker(
				s2.LatLngFromDegrees(rte.Points[0].Lat, rte.Points[0].Lon),
				color.RGBA{0xff, 0, 0, 0xff},
				10.0,
			),
		)
		ctx.AddMarker(
			sm.NewMarker(
				s2.LatLngFromDegrees(rte.Points[len(rte.Points)-1].Lat, rte.Points[len(rte.Points)-1].Lon),
				color.RGBA{0xff, 0, 0, 0xff},
				10.0,
			),
		)

		var dist float64
		var segments []s2.LatLng
		for i, v := range rte.Points {
			ll := s2.LatLngFromDegrees(v.Lat, v.Lon)
			segments = append(segments, ll)
			if i > 0 {
				prev := rte.Points[i-1]
				dist += distance(prev.Lat, prev.Lon, v.Lat, v.Lon)
			}
		}
		ctx.AddPath(sm.NewPath(segments, color.RGBA{0xff, 0, 0, 0xff}, 1.0))

		img, err := ctx.Render()
		if err != nil {
			return err
		}

		buf := &bytes.Buffer{}
		if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: 90}); err != nil {
			return err
		}

		if err := ioutil.WriteFile(fmt.Sprintf("%s.jpg", rte.Name), buf.Bytes(), 0666); err != nil {
			return err
		}
	}
	return nil
}
