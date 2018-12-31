# Simple Serialize (SSZ)

This package implements simple serialize algorithm specified in official Ethereum 2.0 [spec](https://github.com/ethereum/eth2.0-specs/blob/master/specs/simple-serialize.md).

## Interface

### Encodable
A type is Encodable if it implements `EncodeSSZ` and `EncodeSSZSize` function.

```go
type Encodable interface {
	EncodeSSZ(io.Writer) error
	// Estimate the encoding size of the object without doing the actual encoding
	EncodeSSZSize() (uint32, error)
}
```

### Decodable
A type is Decodable if it implements `DecodeSSZ()`.
```go
type Decodable interface {
	DecodeSSZ(io.Reader) error
}
```

### Hashable
A type is Hashable if it implements `TreeHashSSZ()`.
```go
type Hashable interface {
	TreeHashSSZ() ([32]byte, error)
}
```

## API

### Encoding function

```go
// Encode val and output the result into w.
func Encode(w io.Writer, val interface{}) error
```

```go
// EncodeSize returns the target encoding size without doing the actual encoding.
// This is an optional pass. You don't need to call this before the encoding unless you
// want to know the output size first.
func EncodeSize(val interface{}) (uint32, error)
```

### Decoding function
```go
// Decode data read from r and output it into the object pointed by pointer val.
func Decode(r io.Reader, val interface{}) error
```

### Hashing function
```go
// Tree-hash data into [32]byte
func TreeHash(val interface{}) ([32]byte, error)
````

## Usage

Say you have a struct like this
```go
type exampleStruct1 struct {
	Field1 uint8
	Field2 []byte
}
````

You implement the `Encoding` interface for it:

```go
func (e *exampleStruct1) EncodeSSZ(w io.Writer) error {
	return Encode(w, *e)
}

func (e *exampleStruct1) EncodeSSZSize() (uint32, error) {
	return EncodeSize(*e)
}
```

Now you can encode this object like this
```go
e1 := &exampleStruct1{
    Field1: 10,
    Field2: []byte{1, 2, 3, 4},
}
wBuf := new(bytes.Buffer)
if err = e1.EncodeSSZ(wBuf); err != nil {
    return fmt.Errorf("failed to encode: %v", err)
}
encoding := wBuf.Bytes() // encoding becomes [0 0 0 9 10 0 0 0 4 1 2 3 4]
```

You can also get the estimated encoding size
```go
var encodeSize uint32
if encodeSize, err = e1.EncodeSSZSize(); err != nil {
    return fmt.Errorf("failed to get encode size: %v", err)
}
// encodeSize becomes 13
```

To calculate tree-hash of the object
```go
var hash [32]byte
if hash, err = e1.TreeHashSSZ(); err != nil {
    return fmt.Errorf("failed to hash: %v", err)
}
// hash stores the hashing result
```

Similarly, you can implement the `Decodable` interface for this struct

```go
func (e *exampleStruct1) DecodeSSZ(r io.Reader) error {
	return Decode(r, e)
}
```

Now you can decode to create new struct

```go
e2 := new(exampleStruct1)
rBuf := bytes.NewReader(encoding)
if err = e2.DecodeSSZ(rBuf); err != nil {
    return fmt.Errorf("failed to decode: %v", err)
}
// e2 now has the same content as e1
```

## Notes

### Supported data types
- uint8
- uint16
- uint32
- uint64
- slice
- array
- struct
- pointer (nil pointer is not supported)
