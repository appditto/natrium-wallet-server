package models

type DolarsiResponse []struct {
	Casa struct {
		Compra      string  `json:"compra,omitempty"`
		Venta       string  `json:"venta,omitempty"`
		Agencia     string  `json:"agencia,omitempty"`
		Nombre      string  `json:"nombre,omitempty"`
		Variacion   *string `json:"variacion,omitempty"`
		VentaCero   *string `json:"ventaCero,omitempty"`
		Decimales   *string `json:"decimales,omitempty"`
		MejorCompra *string `json:"mejor_compra,omitempty"`
		MejorVenta  *string `json:"mejor_venta,omitempty"`
		Fecha       *string `json:"fecha,omitempty"`
		Recorrido   *string `json:"recorrido,omitempty"`
		Afluencia   *struct {
		} `json:"afluencia,omitempty"`
		Observaciones *struct {
		} `json:"observaciones,omitempty"`
	} `json:"casa,omitempty"`
}
