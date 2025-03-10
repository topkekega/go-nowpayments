package payments

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/matm/go-nowpayments/core"
	"github.com/rotisserie/eris"
)

// PaymentAmount defines common fields used in PaymentArgs and
// Payment structs.
type PaymentAmount struct {
	PriceAmount      float64 `json:"price_amount"`
	PriceCurrency    string  `json:"price_currency"`
	PayCurrency      string  `json:"pay_currency,omitempty"`
	CallbackURL      string  `json:"ipn_callback_url,omitempty"`
	OrderID          string  `json:"order_id,omitempty"`
	OrderDescription string  `json:"order_description,omitempty"`
}

// PaymentArgs are the arguments used to make a payment.
type PaymentArgs struct {
	PaymentAmount

	// FeePaidByUser is optional, required for fixed-rate exchanges with all fees paid by users.
	FeePaidByUser bool `json:"is_fee_paid_by_user,omitempty"`
	// FixedRate is optional, required for fixed-rate exchanges.
	FixedRate bool `json:"fixed_rate,omitempty"`
	// PayoutAddress is optional, usually the funds will go to the address you specify in
	// your personal account. In case you want to receive funds on another address, you can specify
	// it in this parameter.
	PayoutAddress string `json:"payout_address,omitempty"`
	// PayAmount is optional, the amount that users have to pay for the order stated in crypto.
	// You can either specify it yourself, or we will automatically convert the amount indicated
	// in price_amount.
	PayAmount float64 `json:"pay_amount,omitempty"`
	// PayoutCurrency for the cryptocurrency name.
	PayoutCurrency string `json:"payout_currency,omitempty"`
	// PayoutExtraID is optional, extra id or memo or tag for external payout_address.
	PayoutExtraID string `json:"payout_extra_id,omitempty"`
	// PurchaseID is optional, id of purchase for which you want to create another
	// payment, only used for several payments for one order.
	PurchaseID string `json:"purchase_id,omitempty"`
	// optional, case which you want to test (sandbox only).
	Case string `json:"case,omitempty"`
}

// Payment holds payment related information once we get a response
// from the server.
// FIXME: the API doc misses information about returned fields.
// Misses also HTTP return codes.
// Why is purchase_id an int instead of a string (payment status response)?
// Another inconsistency: list of all payments returns a payment ID as an int instead of a string
// https://documenter.getpostman.com/view/7907941/S1a32n38?version=latest#5e37f3ad-0fa1-4292-af51-5c7f95730486
type Payment struct {
	PaymentAmount

	ID                     string  `json:"payment_id"`
	AmountReceived         float64 `json:"amount_received"`
	BurningPercent         int     `json:"burning_percent"`
	CreatedAt              string  `json:"created_at"`
	ExpirationEstimateDate string  `json:"expiration_estimate_date"`
	Network                string  `json:"network"`
	NetworkPrecision       int     `json:"network_precision"`
	PayAddress             string  `json:"pay_address"`
	PayAmount              float64 `json:"pay_amount"`
	PayCurrency            string  `json:"pay_currency"`
	PayinExtraID           string  `json:"payin_extra_id"`
	PurchaseID             string  `json:"purchase_id"`
	SmartContract          string  `json:"smart_contract"`
	Status                 string  `json:"payment_status"`
	TimeLimit              string  `json:"time_limit"`
	UpdatedAt              string  `json:"updated_at"`
}

// UnmarshalJSON provides custom unmarshalling to the Payment struct so it
// can work it all known cases.
// This is to prevent 2 inconsistencies where their API returns:
// ID as an int (after "list payments" call) or a string (after "create payment" call)
// PayAmount as a string or a float64 (difference betwwen prod and sandbox APIs).
func (p *Payment) UnmarshalJSON(b []byte) error {
	type sp struct {
		PaymentAmount

		ID                     interface{} `json:"payment_id"`
		AmountReceived         float64     `json:"amount_received"`
		BurningPercent         int         `json:"burning_percent"`
		CreatedAt              string      `json:"created_at"`
		ExpirationEstimateDate string      `json:"expiration_estimate_date"`
		Network                string      `json:"network"`
		NetworkPrecision       int         `json:"network_precision"`
		PayAddress             string      `json:"pay_address"`
		PayAmount              interface{} `json:"pay_amount"`
		PayCurrency            string      `json:"pay_currency"`
		PayinExtraID           string      `json:"payin_extra_id"`
		PurchaseID             string      `json:"purchase_id"`
		SmartContract          string      `json:"smart_contract"`
		Status                 string      `json:"payment_status"`
		TimeLimit              string      `json:"time_limit"`
		UpdatedAt              string      `json:"updated_at"`
	}
	j := sp{}
	err := json.Unmarshal(b, &j)
	if err != nil {
		return eris.Wrap(err, "payment custom unmarshal")
	}
	z := Payment{
		PaymentAmount:          j.PaymentAmount,
		AmountReceived:         j.AmountReceived,
		BurningPercent:         j.BurningPercent,
		CreatedAt:              j.CreatedAt,
		ExpirationEstimateDate: j.ExpirationEstimateDate,
		Network:                j.Network,
		NetworkPrecision:       j.NetworkPrecision,
		PayAddress:             j.PayAddress,
		PayCurrency:            j.PayCurrency,
		PayinExtraID:           j.PayinExtraID,
		PurchaseID:             j.PurchaseID,
		SmartContract:          j.SmartContract,
		Status:                 j.Status,
		TimeLimit:              j.TimeLimit,
		UpdatedAt:              j.UpdatedAt,
	}
	switch j.PayAmount.(type) {
	case string:
		pa, err := strconv.ParseFloat(j.PayAmount.(string), 64)
		if err != nil {
			return eris.Wrap(err, "parsing pay_amount as a float")
		}
		z.PayAmount = pa
	case float64:
		z.PayAmount = j.PayAmount.(float64)
	default:
		// Any other type (including nil) converts to a zero value,
		// which is the default. Do nothing.
	}
	switch j.ID.(type) {
	case string:
		z.ID = j.ID.(string)
	case float64:
		z.ID = fmt.Sprintf("%d", int(j.ID.(float64)))
	default:
		// Any other type converts to the default value for the type.
		// Do nothing.
	}
	*p = z
	return nil
}

// New creates a payment.
func New(pa *PaymentArgs) (*Payment, error) {
	if pa == nil {
		return nil, errors.New("nil payment args")
	}
	d, err := json.Marshal(pa)
	if err != nil {
		return nil, eris.Wrap(err, "payment args")
	}
	p := &Payment{}
	par := &core.SendParams{
		RouteName: "payment-create",
		Into:      &p,
		Body:      strings.NewReader(string(d)),
	}
	err = core.HTTPSend(par)
	if err != nil {
		return nil, err
	}
	return p, nil
}

type InvoicePaymentArgs struct {
	InvoiceID        string `json:"iid"`
	PayCurrency      string `json:"pay_currency"`
	PurchaseID       string `json:"purchase_id,omitempty"`
	OrderDescription string `json:"order_description,omitempty"`
	CustomerEmail    string `json:"customer_email,omitempty"`
	PayoutCurrency   string `json:"payout_currency,omitempty"`
	PayoutExtraID    string `json:"payout_extra_id,omitempty"`
	PayoutAddress    string `json:"payout_address,omitempty"`
}

// NewFromInvoice creates a payment from an existing invoice. ID is the
// invoice's identifier.
func NewFromInvoice(ipa *InvoicePaymentArgs) (*Payment, error) {
	if ipa == nil {
		return nil, errors.New("nil invoice payment args")
	}
	d, err := json.Marshal(ipa)
	if err != nil {
		return nil, eris.Wrap(err, "payment from invoice args")
	}
	p := &Payment{}
	par := &core.SendParams{
		RouteName: "invoice-payment",
		Into:      &p,
		Body:      strings.NewReader(string(d)),
	}
	err = core.HTTPSend(par)
	if err != nil {
		return nil, err
	}
	return p, nil
}
