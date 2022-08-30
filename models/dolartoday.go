package models

type DolarTodayResponse struct {
	Antibloqueo struct {
		Mobile                   string `json:"mobile"`
		Video                    string `json:"video"`
		CortoAlternativo         string `json:"corto_alternativo"`
		EnableIads               string `json:"enable_iads"`
		EnableAdmobbanners       string `json:"enable_admobbanners"`
		EnableAdmobinterstitials string `json:"enable_admobinterstitials"`
		Alternativo              string `json:"alternativo"`
		Alternativo2             string `json:"alternativo2"`
		Notifications            string `json:"notifications"`
		ResourceID               string `json:"resource_id"`
	} `json:"_antibloqueo"`
	Labels struct {
		A  string `json:"a"`
		A1 string `json:"a1"`
		B  string `json:"b"`
		C  string `json:"c"`
		D  string `json:"d"`
		E  string `json:"e"`
	} `json:"_labels"`
	Timestamp struct {
		Epoch       string `json:"epoch"`
		Fecha       string `json:"fecha"`
		FechaCorta  string `json:"fecha_corta"`
		FechaCorta2 string `json:"fecha_corta2"`
		FechaNice   string `json:"fecha_nice"`
		Dia         string `json:"dia"`
		DiaCorta    string `json:"dia_corta"`
	} `json:"_timestamp"`
	Usd struct {
		Transferencia   float64 `json:"transferencia"`
		TransferCucuta  float64 `json:"transfer_cucuta"`
		Efectivo        float64 `json:"efectivo"`
		EfectivoReal    float64 `json:"efectivo_real"`
		EfectivoCucuta  float64 `json:"efectivo_cucuta"`
		Promedio        float64 `json:"promedio"`
		PromedioReal    float64 `json:"promedio_real"`
		Cencoex         float64 `json:"cencoex"`
		Sicad1          float64 `json:"sicad1"`
		Sicad2          float64 `json:"sicad2"`
		BitcoinRef      float64 `json:"bitcoin_ref"`
		LocalbitcoinRef float64 `json:"localbitcoin_ref"`
		Dolartoday      float64 `json:"dolartoday"`
	} `json:"USD"`
	Eur struct {
		Transferencia  float64 `json:"transferencia"`
		TransferCucuta float64 `json:"transfer_cucuta"`
		Efectivo       float64 `json:"efectivo"`
		EfectivoReal   float64 `json:"efectivo_real"`
		EfectivoCucuta float64 `json:"efectivo_cucuta"`
		Promedio       float64 `json:"promedio"`
		PromedioReal   float64 `json:"promedio_real"`
		Cencoex        float64 `json:"cencoex"`
		Sicad1         float64 `json:"sicad1"`
		Sicad2         float64 `json:"sicad2"`
		Dolartoday     float64 `json:"dolartoday"`
	} `json:"EUR"`
	Col struct {
		Efectivo float64 `json:"efectivo"`
		Transfer float64 `json:"transfer"`
		Compra   float64 `json:"compra"`
		Venta    float64 `json:"venta"`
	} `json:"COL"`
	Gold struct {
		Rate int `json:"rate"`
	} `json:"GOLD"`
	Usdvef struct {
		Rate int `json:"rate"`
	} `json:"USDVEF"`
	Usdcol struct {
		Setfxsell     float64 `json:"setfxsell"`
		Setfxbuy      float64 `json:"setfxbuy"`
		Rate          float64 `json:"rate"`
		Ratecash      float64 `json:"ratecash"`
		Ratetrm       float64 `json:"ratetrm"`
		Trmfactor     float64 `json:"trmfactor"`
		Trmfactorcash float64 `json:"trmfactorcash"`
	} `json:"USDCOL"`
	Eurusd struct {
		Rate int `json:"rate"`
	} `json:"EURUSD"`
	Bcv struct {
		Fecha     string `json:"fecha"`
		FechaNice string `json:"fecha_nice"`
		Liquidez  string `json:"liquidez"`
		Reservas  string `json:"reservas"`
	} `json:"BCV"`
	Misc struct {
		Petroleo string `json:"petroleo"`
		Reservas string `json:"reservas"`
	} `json:"MISC"`
}
