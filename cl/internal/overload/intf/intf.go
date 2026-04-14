package intf

const (
	XGoPackage = true
)

// XGo method overloads for Sprite interface
const (
	XGoo_Sprite_Glide    = ".GlideToTarget,.GlideToXYpos"
	XGoo_Sprite_StepTo   = ".StepToTarget,.StepToXYpos"
	XGoo_Sprite_TurnTo   = ".TurnToDir,.TurnToTarget"
	XGoo_Sprite_SetLayer = ".SetLayerTo,.ChangeLayer"
	XGoo_Sprite_Quote    = ".QuoteMsg,.QuoteMsgEx"
)

type Target = any

type Direction = float64

type Seconds = float64

const (
	XGou_Seconds = "s=1,ms=0.001"
)

type MotionOptions struct {
	Speed     float64
	Animation string
}

type layerAction int

type dirAction int

type Sprite interface {
	GlideToTarget(target Target, secs Seconds)
	GlideToXYpos(x, y float64, secs Seconds)

	StepToTarget(target Target, __xgo_optional_opts *MotionOptions)
	StepToXYpos(x, y float64, __xgo_optional_opts *MotionOptions)

	TurnToDir(dir Direction, __xgo_optional_opts *MotionOptions)
	TurnToTarget(target Target, __xgo_optional_opts *MotionOptions)

	SetLayerTo(layer layerAction)
	ChangeLayer(dir dirAction, delta int)

	QuoteMsg(message string, __xgo_optional_secs Seconds)
	QuoteMsgEx(message, description string, __xgo_optional_secs Seconds)
}
