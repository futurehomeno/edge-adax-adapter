package model

type Login struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	Encrypted bool   `json:"encrypted"`
}

type SetTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Encrypted    bool   `json:"encrypted"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

type AuthStatus struct {
	Status    string `json:"status"`
	ErrorText string `json:"error_text"`
	ErrorCode string `json:"error_code"`
}
