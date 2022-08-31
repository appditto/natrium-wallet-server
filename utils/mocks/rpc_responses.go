package mocks

import (
	"bytes"
	"io"
)

var AccountInfoResponse = io.NopCloser(bytes.NewReader([]byte(
	"{\n    \"frontier\": \"80A6745762493FA21A22718ABFA4F635656A707B48B3324198AC7F3938DE6D4F\",\n    \"open_block\": \"0E3F07F7F2B8AEDEA4A984E29BFE1E3933BA473DD3E27C662EC041F6EA3917A0\",\n    \"representative_block\": \"80A6745762493FA21A22718ABFA4F635656A707B48B3324198AC7F3938DE6D4F\",\n    \"balance\": \"11999999999999999918751838129509869131\",\n    \"confirmed_balance\": \"11999999999999999918751838129509869131\",\n    \"modified_timestamp\": \"1606934662\",\n    \"block_count\": \"22966\",\n    \"account_version\": \"1\",\n    \"confirmed_height\": \"22966\",\n    \"confirmed_frontier\": \"80A6745762493FA21A22718ABFA4F635656A707B48B3324198AC7F3938DE6D4F\",\n    \"representative\": \"nano_1gyeqc6u5j3oaxbe5qy1hyz3q745a318kh8h9ocnpan7fuxnq85cxqboapu5\",\n    \"confirmed_representative\": \"nano_1gyeqc6u5j3oaxbe5qy1hyz3q745a318kh8h9ocnpan7fuxnq85cxqboapu5\",\n    \"weight\": \"11999999999999999918751838129509869131\",\n    \"pending\": \"0\",\n    \"receivable\": \"0\",\n    \"confirmed_pending\": \"0\",\n    \"confirmed_receivable\": \"0\"\n}")))

var ReceivableResponse = io.NopCloser(bytes.NewReader([]byte("{\n  \"blocks\" : {\n    \"000D1BAEC8EC208142C99059B393051BAC8380F9B5A2E6B2489A277D81789F3F\": \"6000000000000000000000000000000\"\n  }\n}")))

var BlockInfoResponse = io.NopCloser(bytes.NewReader([]byte("{\n  \"block_account\": \"nano_1ipx847tk8o46pwxt5qjdbncjqcbwcc1rrmqnkztrfjy5k7z4imsrata9est\",\n  \"amount\": \"30000000000000000000000000000000000\",\n  \"balance\": \"5606157000000000000000000000000000000\",\n  \"height\": \"58\",\n  \"local_timestamp\": \"0\",\n  \"successor\": \"8D3AB98B301224253750D448B4BD997132400CEDD0A8432F775724F2D9821C72\",\n  \"confirmed\": \"true\",\n  \"contents\": {\n    \"type\": \"state\",\n    \"account\": \"nano_1ipx847tk8o46pwxt5qjdbncjqcbwcc1rrmqnkztrfjy5k7z4imsrata9est\",\n    \"previous\": \"CE898C131AAEE25E05362F247760F8A3ACF34A9796A5AE0D9204E86B0637965E\",\n    \"representative\": \"nano_1stofnrxuz3cai7ze75o174bpm7scwj9jn3nxsn8ntzg784jf1gzn1jjdkou\",\n    \"balance\": \"5606157000000000000000000000000000000\",\n    \"link\": \"5D1AA8A45F8736519D707FCB375976A7F9AF795091021D7E9C7548D6F45DD8D5\",\n    \"link_as_account\": \"nano_1qato4k7z3spc8gq1zyd8xeqfbzsoxwo36a45ozbrxcatut7up8ohyardu1z\",\n    \"signature\": \"82D41BC16F313E4B2243D14DFFA2FB04679C540C2095FEE7EAE0F2F26880AD56DD48D87A7CC5DD760C5B2D76EE2C205506AA557BF00B60D8DEE312EC7343A501\",\n    \"work\": \"8a142e07a10996d5\"\n  },\n  \"subtype\": \"send\"\n}")))

var WorkGenerateResponse = io.NopCloser(bytes.NewReader([]byte(
	"{\n  \"work\": \"2b3d689bbcb21dca\",\n  \"difficulty\": \"fffffff93c41ec94\",\n  \"multiplier\": \"1.182623871097636\",\n  \"hash\": \"718CC2121C3E641059BC1C2CFC45666C99E8AE922F7A807B7D07B62C995D79E2\"\n}")))

var AccountBalanceResponse = io.NopCloser(bytes.NewReader([]byte("{\n  \"balance\": \"10000\",\n  \"pending\": \"10000\",\n  \"receivable\": \"10000\"\n}")))
