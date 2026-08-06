package go_ora

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"

	"github.com/sijms/go-ora/v2/configurations"
	"github.com/sijms/go-ora/v2/network"
	"github.com/sijms/go-ora/v2/network/security"
)

const stuhelperOraclePasswordVerifier = 18453

func validateStuHelperOraclePasswordVerifier(verifierType int) error {
	if verifierType != stuhelperOraclePasswordVerifier {
		return errors.New("StuHelper requires an Oracle 12c or newer PBKDF2 password verifier")
	}
	return nil
}

// E infront of the variable means encrypted
type AuthObject struct {
	EServerSessKey   string
	EClientSessKey   string
	EPassword        string
	ESpeedyKey       string
	ServerSessKey    []byte
	ClientSessKey    []byte
	KeyHash          []byte
	Salt             string
	pbkdf2ChkSalt    string
	pbkdf2VgenCount  int
	pbkdf2SderCount  int
	globalUniqueDBID string
	usePadding       bool
	customHash       bool
	VerifierType     int
	tcpNego          *TCPNego
}

// create authentication object through reading data from network
func newAuthObject(_ string, password string, tcpNego *TCPNego, conn *Connection) (*AuthObject, error) {
	session := conn.session
	ret := new(AuthObject)
	ret.tcpNego = tcpNego
	ret.usePadding = false
	ret.customHash = ret.tcpNego.ServerCompileTimeCaps[4]&32 != 0
	loop := true
	for loop {
		messageCode, err := session.GetByte()
		if err != nil {
			return nil, err
		}
		switch messageCode {
		case 8:
			dictLen, err := session.GetInt(4, true, true)
			if err != nil {
				return nil, err
			}
			for x := 0; x < dictLen; x++ {
				key, val, num, err := session.GetKeyVal()
				if err != nil {
					return nil, err
				}
				if bytes.Compare(key, []byte("AUTH_SESSKEY")) == 0 {
					if len(ret.EServerSessKey) == 0 {
						ret.EServerSessKey = string(val)
					}
				} else if bytes.Compare(key, []byte("AUTH_VFR_DATA")) == 0 {
					if len(ret.Salt) == 0 {
						ret.Salt = string(val)
						ret.VerifierType = num
					}
				} else if bytes.Compare(key, []byte("AUTH_PBKDF2_CSK_SALT")) == 0 {
					if len(ret.pbkdf2ChkSalt) == 0 {
						ret.pbkdf2ChkSalt = string(val)
						if len(ret.pbkdf2ChkSalt) != 32 {
							return nil, network.NewOracleError(28041)
						}
					}
				} else if bytes.Compare(key, []byte("AUTH_PBKDF2_VGEN_COUNT")) == 0 {
					if ret.pbkdf2VgenCount == 0 {
						ret.pbkdf2VgenCount, err = strconv.Atoi(string(val))
						if err != nil {
							return nil, network.NewOracleError(28041)
						}
						if ret.pbkdf2VgenCount < 4096 || ret.pbkdf2VgenCount > 100000000 {
							return nil, network.NewOracleError(28041)
						}
					}
				} else if bytes.Compare(key, []byte("AUTH_PBKDF2_SDER_COUNT")) == 0 {
					ret.pbkdf2SderCount, err = strconv.Atoi(string(val))
					if err != nil || ret.pbkdf2SderCount < 3 || ret.pbkdf2SderCount > 100000000 {
						return nil, network.NewOracleError(28041)
					}
				}
			}
		// case 15:
		//	warning, err := network.NewWarningObject(conn.session)
		//	if err != nil {
		//		return nil, err
		//	}
		//	if warning != nil {
		//		fmt.Println(warning)
		//	}
		// case 23:
		//	opCode, err := conn.session.GetByte()
		//	if err != nil {
		//		return nil, err
		//	}
		//	err = conn.getServerNetworkInformation(opCode)
		//	if err != nil {
		//		return nil, err
		//	}
		default:
			err = conn.readMsg(messageCode)
			if err != nil {
				return nil, err
			}
			if messageCode == 4 {
				if session.HasError() {
					return nil, session.GetError()
				}
				loop = false
			}
			// return nil, errors.New(fmt.Sprintf("message code error: received code %d and expected code is 8", messageCode))
		}
	}
	if len(ret.EServerSessKey) != 64 && len(ret.EServerSessKey) != 96 {
		return nil, errors.New("session key should be either 64, 96 bytes long")
	}
	if err := validateStuHelperOraclePasswordVerifier(ret.VerifierType); err != nil {
		return nil, err
	}
	if len(ret.pbkdf2ChkSalt) != 32 || ret.pbkdf2VgenCount < 4096 || ret.pbkdf2SderCount < 3 {
		return nil, network.NewOracleError(28041)
	}
	salt, err := hex.DecodeString(ret.Salt)
	if err != nil {
		return nil, err
	}
	message := append(salt, []byte("AUTH_PBKDF2_SPEEDY_KEY")...)
	speedyKey := generateSpeedyKey(message, []byte(password), ret.pbkdf2VgenCount)

	buffer := append(speedyKey, salt...)
	hash := sha512.New()
	if _, err := hash.Write(buffer); err != nil {
		return nil, err
	}
	key := hash.Sum(nil)[:32]
	const padding = false
	// get the server session key
	ret.ServerSessKey, err = decryptSessionKey(padding, key, ret.EServerSessKey)
	if err != nil {
		return nil, err
	}

	// note if serverSessKey length is less than the expected length according to verifier generate random one
	// generate new key for client
	ret.ClientSessKey = make([]byte, len(ret.ServerSessKey))
	for {
		_, err = rand.Read(ret.ClientSessKey)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(ret.ClientSessKey, ret.ServerSessKey) {
			break
		}
	}

	// encrypt the client key
	ret.EClientSessKey, err = encryptSessionKey(padding, key, ret.ClientSessKey)
	if err != nil {
		return nil, err
	}

	// get the hash key form server and client session key
	newKey, err := ret.generatePasswordEncKey()
	if err != nil {
		return nil, err
	}
	// encrypt the password
	ret.EPassword, err = encryptPassword([]byte(password), newKey, true)
	if err != nil {
		return nil, err
	}
	ret.ESpeedyKey, err = encryptPassword(speedyKey, newKey, padding)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// write authentication data to network
func (obj *AuthObject) Write(connOption *configurations.ConnectionConfig, mode LogonMode, session *network.Session) error {
	keys := make([]string, 0, 20)
	values := make([]string, 0, 20)
	flags := make([]uint8, 0, 20)
	appendKeyVal := func(key, val string, f uint8) {
		keys = append(keys, key)
		values = append(values, val)
		flags = append(flags, f)
	}
	index := 0
	if len(obj.EClientSessKey) > 0 {
		appendKeyVal("AUTH_SESSKEY", obj.EClientSessKey, 1)
		index++
	}
	if len(obj.EPassword) > 0 {
		appendKeyVal("AUTH_PASSWORD", obj.EPassword, 0)
		index++
	}
	if len(obj.ESpeedyKey) > 0 {
		appendKeyVal("AUTH_PBKDF2_SPEEDY_KEY", obj.ESpeedyKey, 0)
		index++
	}
	appendKeyVal("AUTH_TERMINAL", connOption.ClientInfo.HostName, 0)
	index++
	appendKeyVal("AUTH_PROGRAM_NM", connOption.ClientInfo.ProgramName, 0)
	index++
	appendKeyVal("AUTH_MACHINE", connOption.ClientInfo.HostName, 0)
	index++
	appendKeyVal("AUTH_PID", fmt.Sprintf("%d", connOption.ClientInfo.PID), 0)
	index++
	appendKeyVal("AUTH_SID", connOption.ClientInfo.OSUserName, 0)
	index++
	appendKeyVal("AUTH_CONNECT_STRING", connOption.ConnectionData(), 0)
	index++
	appendKeyVal("SESSION_CLIENT_CHARSET", strconv.Itoa(int(obj.tcpNego.ServerCharset)), 0)
	index++
	appendKeyVal("SESSION_CLIENT_LIB_TYPE", "0", 0)
	index++
	appendKeyVal("SESSION_CLIENT_DRIVER_NAME", connOption.ClientInfo.DriverName, 0)
	index++
	appendKeyVal("SESSION_CLIENT_VERSION", "2.0.0.0", 0)
	index++
	appendKeyVal("SESSION_CLIENT_LOBATTR", "1", 0)
	index++
	// StuHelper intentionally omits go-ora's session-mutation login payload. The
	// campus data source policy permits authentication and fixed SELECTs, but no
	// session mutation, even when it would normally be part of login.
	if len(connOption.ProxyClientName) > 0 {
		appendKeyVal("PROXY_CLIENT_NAME", connOption.ProxyClientName, 0)
		index++
	}
	session.ResetBuffer()
	session.PutBytes(3, 0x73, 0)
	if len(connOption.UserID) > 0 {
		session.PutBytes(1)
		session.PutInt(len(connOption.UserID), 4, true, true)
	} else {
		session.PutBytes(0, 0)
	}
	// if proxy auth logonMode |= 0x400
	if len(connOption.UserID) > 0 && len(obj.EPassword) > 0 {
		mode |= UserAndPass
	}
	session.PutUint(int(mode|NoNewPass), 4, true, true)
	session.PutBytes(1)
	session.PutUint(index, 4, true, true)
	session.PutBytes(1, 1)
	if len(connOption.UserID) > 0 {
		session.PutString(connOption.UserID)
	}
	for i := 0; i < index; i++ {
		session.PutKeyValString(keys[i], values[i], flags[i])
	}
	return session.Write()
}

func generateSpeedyKey(buffer, key []byte, turns int) []byte {
	mac := hmac.New(sha512.New, key)
	mac.Write(append(buffer, 0, 0, 0, 1))
	firstHash := mac.Sum(nil)
	tempHash := make([]byte, len(firstHash))
	copy(tempHash, firstHash)
	for index1 := 2; index1 <= turns; index1++ {
		// mac = hmac.New(sha512.New, []byte("ter1234"))
		mac.Reset()
		mac.Write(tempHash)
		tempHash = mac.Sum(nil)
		for index2 := 0; index2 < 64; index2++ {
			firstHash[index2] = firstHash[index2] ^ tempHash[index2]
		}
	}
	return firstHash
}

// decrypt session key that come from the server
func decryptSessionKey(padding bool, encKey []byte, sessionKey string) ([]byte, error) {
	result, err := hex.DecodeString(sessionKey)
	if err != nil {
		return nil, err
	}
	blk, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, err
	}
	//if padding {
	//	result = PKCS5Padding(result, blk.BlockSize())
	//}
	enc := cipher.NewCBCDecrypter(blk, make([]byte, 16))
	output := make([]byte, len(result))
	enc.CryptBlocks(output, result)
	cutLen := 0
	if padding {
		num := int(output[len(output)-1])
		if num < enc.BlockSize() {
			apply := true
			for x := len(output) - num; x < len(output); x++ {
				if output[x] != uint8(num) {
					apply = false
					break
				}
			}
			if apply {
				cutLen = int(output[len(output)-1])
			}
		}
	}
	return output[:len(output)-cutLen], nil
}

// encrypt session key that generated from the client
func encryptSessionKey(padding bool, encKey []byte, sessionKey []byte) (string, error) {
	blk, err := aes.NewCipher(encKey)
	if err != nil {
		return "", err
	}
	enc := cipher.NewCBCEncrypter(blk, make([]byte, 16))
	originalLen := len(sessionKey)
	sessionKey = security.PKCS5Padding(sessionKey, blk.BlockSize())
	//if padding {
	//
	//}
	output := make([]byte, len(sessionKey))
	enc.CryptBlocks(output, sessionKey)
	if !padding {
		return fmt.Sprintf("%X", output[:originalLen]), nil
	}
	return fmt.Sprintf("%X", output), nil

	// cryptoServiceProvider.Mode = CipherMode.CBC;
	// cryptoServiceProvider.KeySize = key.Length * 8;
	// cryptoServiceProvider.BlockSize = O5LogonHelper.d;
	// cryptoServiceProvider.Key = key;
	// cryptoServiceProvider.IV = O5LogonHelper.f;
	// numArray = cryptoServiceProvider.CreateEncryptor().TransformFinalBlock(buffer, 0, buffer.Length);
}

// encrypt user password
func encryptPassword(password, key []byte, padding bool) (string, error) {
	buff1 := make([]byte, 0x10)
	_, err := rand.Read(buff1)
	if err != nil {
		return "", err
	}
	buffer := append(buff1, password...)
	return encryptSessionKey(padding, key, buffer)
}

// generate encryption key for the password this depends on database verifier type
func (obj *AuthObject) generatePasswordEncKey() ([]byte, error) {
	if err := validateStuHelperOraclePasswordVerifier(obj.VerifierType); err != nil {
		return nil, err
	}
	if obj.tcpNego.ServerCompileTimeCaps[4]&32 == 0 {
		return nil, errors.New("Oracle server did not negotiate modern password derivation")
	}
	buffer := append(obj.ClientSessKey, obj.ServerSessKey...)
	keyBuffer := fmt.Sprintf("%X", buffer)
	df2key, err := hex.DecodeString(obj.pbkdf2ChkSalt)
	if err != nil {
		return nil, err
	}
	return generateSpeedyKey(df2key, []byte(keyBuffer), obj.pbkdf2SderCount)[:32], nil
}

//func (obj *AuthObject) VerifyResponse(response string) bool {
//	key, err := decryptSessionKey(true, obj.KeyHash, response)
//	if err != nil {
//		fmt.Println(err)
//		return false
//	}
//	//fmt.Printf("%#v\n", key)
//	return bytes.Compare(key[16:], []byte{83, 69, 82, 86, 69, 82, 95, 84, 79, 95, 67, 76, 73, 69, 78, 84}) == 0
//	//KZSR_SVR_RESPONSE = new byte[16]{ (byte) 83, (byte) 69, (byte) 82, (byte) 86, (byte) 69, (byte) 82, (byte) 95, (byte) 84, (byte) 79,
//	//(byte) 95, (byte) 67, (byte) 76, (byte) 73, (byte) 69, (byte) 78, (byte) 84 };
//
//}

//func (obj *AuthObject) TestResponse(password, pbkdf2ChkSalt string, vGenCount, sDerCount int) error {
//	padding := false
//	obj.pbkdf2ChkSalt = pbkdf2ChkSalt
//	obj.pbkdf2VgenCount = vGenCount
//	obj.pbkdf2SderCount = sDerCount
//	obj.tcpNego = &TCPNego{
//		MessageCode:           0,
//		ProtocolServerVersion: 0,
//		ProtocolServerString:  "",
//		OracleVersion:         0,
//		ServerCharset:         0,
//		ServerFlags:           0,
//		CharsetElem:           0,
//		ServernCharset:        0,
//		ServerCompileTimeCaps: []byte{0, 0, 0, 0, 32},
//		ServerRuntimeCaps:     nil,
//	}
//	salt, err := hex.DecodeString(obj.Salt)
//	if err != nil {
//		return err
//	}
//	message := append(salt, []byte("AUTH_PBKDF2_SPEEDY_KEY")...)
//	speedyKey := generateSpeedyKey(message, []byte(password), obj.pbkdf2VgenCount)
//
//	buffer := append(speedyKey, salt...)
//	hash := sha512.New()
//	hash.Write(buffer)
//	key := hash.Sum(nil)[:32]
//	obj.ServerSessKey, err = decryptSessionKey(padding, key, obj.EServerSessKey)
//	if err != nil {
//		return err
//	}
//	obj.ClientSessKey, err = decryptSessionKey(padding, key, obj.EClientSessKey)
//	if err != nil {
//		return err
//	}
//	newKey, err := obj.generatePasswordEncKey()
//	if err != nil {
//		return err
//	}
//	fmt.Println(decryptSessionKey(padding, newKey, obj.EPassword))
//
//	obj.EPassword, err = encryptPassword([]byte(password), newKey, false)
//	if err != nil {
//		return err
//	}
//	obj.ESpeedyKey, err = encryptPassword(speedyKey, newKey, false)
//	return err
//}
