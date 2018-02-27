// Package ewkb 实现了 扩展的著名二进制编码和解码。
// See https://github.com/postgis/postgis/blob/2.1.0/doc/ZMSgeoms.txt.
//
// 如果你正在将几何图形编码成 EWKB ,并存进 PostgreSQL/PostGIS。你必须设置 binary_parameters=yes，在
//你向 sql.Open传递的数据资源中
package ewkb

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/chengxiaoer/geomGo"
	"github.com/chengxiaoer/geomGo/encoding/wkbcommon"
)

var (
	// XDR is big endian.
	XDR = wkbcommon.XDR
	// NDR is little endian.
	NDR = wkbcommon.NDR
)

const (
	ewkbZ    = 0x80000000
	ewkbM    = 0x40000000
	ewkbSRID = 0x20000000
)

// Read函数 从 r 中读取任意几何图形.
func Read(r io.Reader) (geom.T, error) {

	ewkbByteOrder, err := wkbcommon.ReadByte(r)
	if err != nil {
		return nil, err
	}
	var byteOrder binary.ByteOrder
	switch ewkbByteOrder {
	case wkbcommon.XDRID:
		byteOrder = XDR
	case wkbcommon.NDRID:
		byteOrder = NDR
	default:
		return nil, wkbcommon.ErrUnknownByteOrder(ewkbByteOrder)
	}

	ewkbGeometryType, err := wkbcommon.ReadUInt32(r, byteOrder)
	if err != nil {
		return nil, err
	}
	t := wkbcommon.Type(ewkbGeometryType)

	layout := geom.NoLayout
	switch t & (ewkbZ | ewkbM) {
	case 0:
		layout = geom.XY
	case ewkbZ:
		layout = geom.XYZ
	case ewkbM:
		layout = geom.XYM
	case ewkbZ | ewkbM:
		layout = geom.XYZM
	default:
		return nil, wkbcommon.ErrUnknownType(t)
	}

	var srid uint32
	if ewkbGeometryType&ewkbSRID != 0 {
		srid, err = wkbcommon.ReadUInt32(r, byteOrder)
		if err != nil {
			return nil, err
		}
	}

	switch t &^ (ewkbZ | ewkbM | ewkbSRID) {
	case wkbcommon.PointID:
		flatCoords, err := wkbcommon.ReadFlatCoords0(r, byteOrder, layout.Stride())
		if err != nil {
			return nil, err
		}
		return geom.NewPointFlat(layout, flatCoords).SetSRID(int(srid)), nil
	case wkbcommon.LineStringID:
		flatCoords, err := wkbcommon.ReadFlatCoords1(r, byteOrder, layout.Stride())
		if err != nil {
			return nil, err
		}
		return geom.NewLineStringFlat(layout, flatCoords).SetSRID(int(srid)), nil
	case wkbcommon.PolygonID:
		flatCoords, ends, err := wkbcommon.ReadFlatCoords2(r, byteOrder, layout.Stride())
		if err != nil {
			return nil, err
		}
		return geom.NewPolygonFlat(layout, flatCoords, ends).SetSRID(int(srid)), nil
	case wkbcommon.MultiPointID:
		n, err := wkbcommon.ReadUInt32(r, byteOrder)
		if err != nil {
			return nil, err
		}
		if n > wkbcommon.MaxGeometryElements[1] {
			return nil, wkbcommon.ErrGeometryTooLarge{Level: 1, N: n, Limit: wkbcommon.MaxGeometryElements[1]}
		}
		mp := geom.NewMultiPoint(layout).SetSRID(int(srid))
		for i := uint32(0); i < n; i++ {
			g, err := Read(r)
			if err != nil {
				return nil, err
			}
			p, ok := g.(*geom.Point)
			if !ok {
				return nil, wkbcommon.ErrUnexpectedType{Got: g, Want: &geom.Point{}}
			}
			if err = mp.Push(p); err != nil {
				return nil, err
			}
		}
		return mp, nil
	case wkbcommon.MultiLineStringID:
		n, err := wkbcommon.ReadUInt32(r, byteOrder)
		if err != nil {
			return nil, err
		}
		if n > wkbcommon.MaxGeometryElements[2] {
			return nil, wkbcommon.ErrGeometryTooLarge{Level: 2, N: n, Limit: wkbcommon.MaxGeometryElements[2]}
		}
		mls := geom.NewMultiLineString(layout).SetSRID(int(srid))
		for i := uint32(0); i < n; i++ {
			g, err := Read(r)
			if err != nil {
				return nil, err
			}
			p, ok := g.(*geom.LineString)
			if !ok {
				return nil, wkbcommon.ErrUnexpectedType{Got: g, Want: &geom.LineString{}}
			}
			if err = mls.Push(p); err != nil {
				return nil, err
			}
		}
		return mls, nil
	case wkbcommon.MultiPolygonID:
		n, err := wkbcommon.ReadUInt32(r, byteOrder)
		if err != nil {
			return nil, err
		}
		if n > wkbcommon.MaxGeometryElements[3] {
			return nil, wkbcommon.ErrGeometryTooLarge{Level: 3, N: n, Limit: wkbcommon.MaxGeometryElements[3]}
		}
		mp := geom.NewMultiPolygon(layout).SetSRID(int(srid))
		for i := uint32(0); i < n; i++ {
			g, err := Read(r)
			if err != nil {
				return nil, err
			}
			p, ok := g.(*geom.Polygon)
			if !ok {
				return nil, wkbcommon.ErrUnexpectedType{Got: g, Want: &geom.Polygon{}}
			}
			if err = mp.Push(p); err != nil {
				return nil, err
			}
		}
		return mp, nil
	case wkbcommon.GeometryCollectionID:
		n, err := wkbcommon.ReadUInt32(r, byteOrder)
		if err != nil {
			return nil, err
		}
		if n > wkbcommon.MaxGeometryElements[1] {
			return nil, wkbcommon.ErrGeometryTooLarge{Level: 1, N: n, Limit: wkbcommon.MaxGeometryElements[1]}
		}
		gc := geom.NewGeometryCollection().SetSRID(int(srid))
		for i := uint32(0); i < n; i++ {
			g, err := Read(r)
			if err != nil {
				return nil, err
			}
			if err = gc.Push(g); err != nil {
				return nil, err
			}
		}
		return gc, nil
	default:
		return nil, wkbcommon.ErrUnsupportedType(ewkbGeometryType)
	}

}

// Unmarshal函数  从data []byte中解码任意图形.
func Unmarshal(data []byte) (geom.T, error) {
	return Read(bytes.NewBuffer(data))
}

// Write函数 向 w写入任意的几何图形.
func Write(w io.Writer, byteOrder binary.ByteOrder, g geom.T) error {

	var ewkbByteOrder byte
	switch byteOrder {
	case XDR:
		ewkbByteOrder = wkbcommon.XDRID
	case NDR:
		ewkbByteOrder = wkbcommon.NDRID
	default:
		return wkbcommon.ErrUnsupportedByteOrder{}
	}
	if err := binary.Write(w, byteOrder, ewkbByteOrder); err != nil {
		return err
	}

	var ewkbGeometryType uint32
	switch g.(type) {
	case *geom.Point:
		ewkbGeometryType = wkbcommon.PointID
	case *geom.LineString:
		ewkbGeometryType = wkbcommon.LineStringID
	case *geom.Polygon:
		ewkbGeometryType = wkbcommon.PolygonID
	case *geom.MultiPoint:
		ewkbGeometryType = wkbcommon.MultiPointID
	case *geom.MultiLineString:
		ewkbGeometryType = wkbcommon.MultiLineStringID
	case *geom.MultiPolygon:
		ewkbGeometryType = wkbcommon.MultiPolygonID
	case *geom.GeometryCollection:
		ewkbGeometryType = wkbcommon.GeometryCollectionID
	default:
		return geom.ErrUnsupportedType{Value: g}
	}
	switch g.Layout() {
	case geom.XY:
	case geom.XYZ:
		ewkbGeometryType |= ewkbZ
	case geom.XYM:
		ewkbGeometryType |= ewkbM
	case geom.XYZM:
		ewkbGeometryType |= ewkbZ | ewkbM
	default:
		return geom.ErrUnsupportedLayout(g.Layout())
	}
	srid := g.SRID()
	if srid != 0 {
		ewkbGeometryType |= ewkbSRID
	}
	if err := binary.Write(w, byteOrder, ewkbGeometryType); err != nil {
		return err
	}
	if ewkbGeometryType&ewkbSRID != 0 {
		if err := binary.Write(w, byteOrder, uint32(srid)); err != nil {
			return err
		}
	}

	switch g := g.(type) {
	case *geom.Point:
		return wkbcommon.WriteFlatCoords0(w, byteOrder, g.FlatCoords())
	case *geom.LineString:
		return wkbcommon.WriteFlatCoords1(w, byteOrder, g.FlatCoords(), g.Stride())
	case *geom.Polygon:
		return wkbcommon.WriteFlatCoords2(w, byteOrder, g.FlatCoords(), g.Ends(), g.Stride())
	case *geom.MultiPoint:
		n := g.NumPoints()
		if err := binary.Write(w, byteOrder, uint32(n)); err != nil {
			return err
		}
		for i := 0; i < n; i++ {
			if err := Write(w, byteOrder, g.Point(i)); err != nil {
				return err
			}
		}
		return nil
	case *geom.MultiLineString:
		n := g.NumLineStrings()
		if err := binary.Write(w, byteOrder, uint32(n)); err != nil {
			return err
		}
		for i := 0; i < n; i++ {
			if err := Write(w, byteOrder, g.LineString(i)); err != nil {
				return err
			}
		}
		return nil
	case *geom.MultiPolygon:
		n := g.NumPolygons()
		if err := binary.Write(w, byteOrder, uint32(n)); err != nil {
			return err
		}
		for i := 0; i < n; i++ {
			if err := Write(w, byteOrder, g.Polygon(i)); err != nil {
				return err
			}
		}
		return nil
	case *geom.GeometryCollection:
		n := g.NumGeoms()
		if err := binary.Write(w, byteOrder, uint32(n)); err != nil {
			return err
		}
		for i := 0; i < n; i++ {
			if err := Write(w, byteOrder, g.Geom(i)); err != nil {
				return err
			}
		}
		return nil
	default:
		return geom.ErrUnsupportedType{Value: g}
	}

}

// Marshal函数 向 a []byte中编码任意几何类型.
func Marshal(g geom.T, byteOrder binary.ByteOrder) ([]byte, error) {
	w := bytes.NewBuffer(nil)
	if err := Write(w, byteOrder, g); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}
