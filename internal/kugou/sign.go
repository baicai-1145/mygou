package kugou

import (
	"sort"
	"strconv"
	"strings"
)

// SignatureAndroid is KuGouMusicApi's signatureAndroidParams for the lite
// platform: MD5(salt + sorted key=value pairs + body + salt).
func SignatureAndroid(params map[string]string, data string) string {
	return md5Hex(androidSalt + sortedParams(params) + data + androidSalt)
}

// ParamsKey is KuGouMusicApi's signParamsKey: the key some JSON endpoints
// ask for, made of the app id, the salt and the client version.
func ParamsKey(ts string) string {
	return md5Hex(strconv.Itoa(AppID) + androidSalt + strconv.Itoa(ClientVer) + ts)
}

// SignatureWeb is the web client's signature, used by the login endpoints.
func SignatureWeb(params map[string]string, data string) string {
	return md5Hex(webSalt + sortedParams(params) + data + webSalt)
}

// SignatureRegister signs a device registration: the sorted values alone,
// between "1014".
func SignatureRegister(params map[string]string) string {
	values := make([]string, 0, len(params))
	for _, v := range params {
		values = append(values, v)
	}
	sort.Strings(values)
	return md5Hex("1014" + strings.Join(values, "") + "1014")
}

// SignKey is the key parameter of endpoints that ask for one, as /song/url
// does.
func SignKey(hash, mid string, userid int64, appid string) string {
	if appid == "" {
		appid = "3116"
	}
	uid := "0"
	if userid != 0 {
		uid = itoa(userid)
	}
	return md5Hex(hash + signKeySalt + appid + mid + uid)
}
