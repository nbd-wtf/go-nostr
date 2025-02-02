package nip60

import "github.com/elnosh/gonuts/cashu"

func GetProofsAndMint(tokenStr string) (cashu.Proofs, string, error) {
	token, err := cashu.DecodeToken(tokenStr)
	if err != nil {
		return nil, "", err
	}
	return token.Proofs(), token.Mint(), nil
}

func MakeTokenString(proofs cashu.Proofs, mint string) string {
	token, err := cashu.NewTokenV4(proofs, mint, cashu.Sat, true)
	if err != nil {
		panic(err)
	}

	tokenStr, err := token.Serialize()
	if err != nil {
		panic(err)
	}

	return tokenStr
}
