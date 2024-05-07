package experiment

type BlockHasBody[T any] interface {
	Body() T
}

func GetBody[B BlockHasBody[T], T UpToAltair](block B) Altair {
	return block.Body()
}

type Phase0Body struct{}

func (b *Phase0Body) Phase0Attr() {}

type AltairBody struct {
	Phase0Body
}

func (b *AltairBody) Phase0Attr() {}
func (b *AltairBody) AltairAttr() {}

type Phase0 interface {
	Phase0Attr()
}

type Altair interface {
	Phase0
	AltairAttr()
}

type UpToAltair interface {
	*Phase0Body | *AltairBody
}

type Block[T UpToAltair] struct {
	body T
}

func (b *Block[T]) Body() T {
	return b.body
}

func processAltair[B BlockHasBody[T], T UpToAltair](block B) {
	body := GetBody[B, T](block)
	uta := body.(Altair)
	body.Phase0Attr()
}

/*
type UnionBlock struct {
	body AltairBody
}

func (b *UnionBlock) Body() AltairBody {
	return b.body
}

type Phase0Body interface {
	A()
}

type AltairBody interface {
	Phase0Body
	B()
}

type unionBody struct {
}

func (u *unionBody) A() {}
func (u *unionBody) B() {}

func NeedsAltair[T BlockHasBody[AltairBody]](block T) {
	body := block.Body()
	body.B()
}

type BlockHasAltair interface {
	BlockHasBody[Phase0Body]
}

func NeedsPhase0[T BlockHasAltair](block T) {
	body := GetBody(block)
	body.A()
}

func TestNeedsPhase0() {
	//NeedsPhase0(&UnionBlock{body: &unionBody{}})
	body := GetBody(&UnionBlock{body: &unionBody{}})
}

*/
