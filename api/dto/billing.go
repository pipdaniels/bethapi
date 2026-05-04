package dto

type TopupRequest struct {
	Amount   float64 `json:"amount" form:"amount" validate:"required,gt=0"`
	Currency string  `json:"currency" form:"currency" validate:"required,oneof=USD NGN KES GHS"`
}

type SubscribeRequest struct {
	Plan     string `json:"plan" form:"plan" validate:"required,oneof=pro ultra"`
	Currency string `json:"currency" form:"currency" validate:"required,oneof=USD NGN KES GHS"`
}

type CheckoutResponse struct {
	URL string `json:"url"`
}
