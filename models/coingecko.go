package models

import "time"

type CoingeckoResponse struct {
	ID              string      `json:"id"`
	Symbol          string      `json:"symbol"`
	Name            string      `json:"name"`
	AssetPlatformID interface{} `json:"asset_platform_id"`
	Platforms       struct {
		NAMING_FAILED string `json:""`
	} `json:"platforms"`
	BlockTimeInMinutes int           `json:"block_time_in_minutes"`
	HashingAlgorithm   string        `json:"hashing_algorithm"`
	Categories         []string      `json:"categories"`
	PublicNotice       string        `json:"public_notice"`
	AdditionalNotices  []interface{} `json:"additional_notices"`
	Description        struct {
		En string `json:"en"`
	} `json:"description"`
	Links struct {
		Homepage                    []string `json:"homepage"`
		BlockchainSite              []string `json:"blockchain_site"`
		OfficialForumURL            []string `json:"official_forum_url"`
		ChatURL                     []string `json:"chat_url"`
		AnnouncementURL             []string `json:"announcement_url"`
		TwitterScreenName           string   `json:"twitter_screen_name"`
		FacebookUsername            string   `json:"facebook_username"`
		BitcointalkThreadIdentifier int      `json:"bitcointalk_thread_identifier"`
		TelegramChannelIdentifier   string   `json:"telegram_channel_identifier"`
		SubredditURL                string   `json:"subreddit_url"`
		ReposURL                    struct {
			Github    []string      `json:"github"`
			Bitbucket []interface{} `json:"bitbucket"`
		} `json:"repos_url"`
	} `json:"links"`
	Image struct {
		Thumb string `json:"thumb"`
		Small string `json:"small"`
		Large string `json:"large"`
	} `json:"image"`
	CountryOrigin                string      `json:"country_origin"`
	GenesisDate                  interface{} `json:"genesis_date"`
	SentimentVotesUpPercentage   float64     `json:"sentiment_votes_up_percentage"`
	SentimentVotesDownPercentage float64     `json:"sentiment_votes_down_percentage"`
	MarketCapRank                int         `json:"market_cap_rank"`
	CoingeckoRank                int         `json:"coingecko_rank"`
	CoingeckoScore               float64     `json:"coingecko_score"`
	DeveloperScore               float64     `json:"developer_score"`
	CommunityScore               float64     `json:"community_score"`
	LiquidityScore               float64     `json:"liquidity_score"`
	PublicInterestScore          float64     `json:"public_interest_score"`
	MarketData                   struct {
		CurrentPrice                           map[string]float64 `json:"current_price"`
		TotalValueLocked                       interface{}        `json:"total_value_locked"`
		McapToTvlRatio                         interface{}        `json:"mcap_to_tvl_ratio"`
		FdvToTvlRatio                          interface{}        `json:"fdv_to_tvl_ratio"`
		Roi                                    interface{}        `json:"roi"`
		Ath                                    map[string]float64 `json:"ath"`
		AthChangePercentage                    map[string]float64 `json:"ath_change_percentage"`
		AthDate                                map[string]string  `json:"ath_date"`
		Atl                                    map[string]float64 `json:"atl"`
		AtlChangePercentage                    map[string]float64 `json:"atl_change_percentage"`
		AtlDate                                map[string]string  `json:"atl_date"`
		MarketCap                              map[string]float64 `json:"market_cap"`
		MarketCapRank                          int                `json:"market_cap_rank"`
		FullyDilutedValuation                  map[string]float64 `json:"fully_diluted_valuation"`
		TotalVolume                            map[string]float64 `json:"total_volume"`
		High24H                                map[string]float64 `json:"high_24h"`
		Low24H                                 map[string]float64 `json:"low_24h"`
		PriceChange24H                         float64            `json:"price_change_24h"`
		PriceChangePercentage24H               float64            `json:"price_change_percentage_24h"`
		PriceChangePercentage7D                float64            `json:"price_change_percentage_7d"`
		PriceChangePercentage14D               float64            `json:"price_change_percentage_14d"`
		PriceChangePercentage30D               float64            `json:"price_change_percentage_30d"`
		PriceChangePercentage60D               float64            `json:"price_change_percentage_60d"`
		PriceChangePercentage200D              float64            `json:"price_change_percentage_200d"`
		PriceChangePercentage1Y                float64            `json:"price_change_percentage_1y"`
		MarketCapChange24H                     int                `json:"market_cap_change_24h"`
		MarketCapChangePercentage24H           float64            `json:"market_cap_change_percentage_24h"`
		PriceChange24HInCurrency               map[string]float64 `json:"price_change_24h_in_currency"`
		PriceChangePercentage1HInCurrency      map[string]float64 `json:"price_change_percentage_1h_in_currency"`
		PriceChangePercentage24HInCurrency     map[string]float64 `json:"price_change_percentage_24h_in_currency"`
		PriceChangePercentage7DInCurrency      map[string]float64 `json:"price_change_percentage_7d_in_currency"`
		PriceChangePercentage14DInCurrency     map[string]float64 `json:"price_change_percentage_14d_in_currency"`
		PriceChangePercentage30DInCurrency     map[string]float64 `json:"price_change_percentage_30d_in_currency"`
		PriceChangePercentage60DInCurrency     map[string]float64 `json:"price_change_percentage_60d_in_currency"`
		PriceChangePercentage200DInCurrency    map[string]float64 `json:"price_change_percentage_200d_in_currency"`
		PriceChangePercentage1YInCurrency      map[string]float64 `json:"price_change_percentage_1y_in_currency"`
		MarketCapChange24HInCurrency           map[string]float64 `json:"market_cap_change_24h_in_currency"`
		MarketCapChangePercentage24HInCurrency map[string]float64 `json:"market_cap_change_percentage_24h_in_currency"`
		TotalSupply                            float64            `json:"total_supply"`
		MaxSupply                              float64            `json:"max_supply"`
		CirculatingSupply                      float64            `json:"circulating_supply"`
		LastUpdated                            time.Time          `json:"last_updated"`
	} `json:"market_data"`
	PublicInterestStats struct {
		AlexaRank   int         `json:"alexa_rank"`
		BingMatches interface{} `json:"bing_matches"`
	} `json:"public_interest_stats"`
	StatusUpdates []struct {
		Description string    `json:"description"`
		Category    string    `json:"category"`
		CreatedAt   time.Time `json:"created_at"`
		User        string    `json:"user"`
		UserTitle   string    `json:"user_title"`
		Pin         bool      `json:"pin"`
		Project     struct {
			Type   string `json:"type"`
			ID     string `json:"id"`
			Name   string `json:"name"`
			Symbol string `json:"symbol"`
			Image  struct {
				Thumb string `json:"thumb"`
				Small string `json:"small"`
				Large string `json:"large"`
			} `json:"image"`
		} `json:"project"`
	} `json:"status_updates"`
	LastUpdated time.Time `json:"last_updated"`
}
