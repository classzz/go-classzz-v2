package ethash

import (
	crand "crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/classzz/go-classzz-v2/common"
	"github.com/classzz/go-classzz-v2/common/hexutil"
	"github.com/classzz/go-classzz-v2/core/types"
	"github.com/classzz/go-classzz-v2/log"
)

type device struct {
	id              string
	regcount        int
	isMining        bool
	isActive        bool
	isAbort         bool
	nonce           uint64
	currentUnixTime int64
	unixTime        int64
	minerAddr       net.Addr
	RemoteAddr      string
	mu              sync.Mutex
	hash            string
}

type retErr struct {
	Id  string `json:"id"`
	rs  string `json:"result"`
	Err string `json:"error"`
}

type GetworkInfo struct {
	Hash   string `json:"hash"`
	Nonce  uint64 `json:"nonce"`
	Target string `json:"target"`
	Height uint64 `json:"height"`
}

type ResponseGWHeader struct {
	Id  string      `json:"id"`
	Gw  GetworkInfo `json:"result"`
	Err string      `json:"error"`
}

type ResponseSWHeader struct {
	Id  string         `json:"id"`
	Sw  SubmitworkSend `json:"result"`
	Err string         `json:"error"`
}

type SubmitworkSend struct {
	State string `json:"state"`
}

type GetSubmitInfo struct {
	Mac       string `json:"mac"`
	Hash      string `json:"hash"`
	MixDigest string `json:"mixDigest"`
	Nonce     int64  `json:"nonce"`
	Hashrate  uint64 `json:"hashrate"`
}

type GetSubmitInfoAndHead struct {
	Id     string          `json:"id"`
	Method string          `json:"method"`
	Params []GetSubmitInfo `json:"params"`
}

type remoteSealerServer struct {
	mu            sync.Mutex
	ethash        *Ethash
	api           *API
	httpStartFlag int
	regmu         sync.Mutex
	devices       map[string]*device
	requestExit   chan struct{}
}

func newRemoteSealer(ethash *Ethash) *remoteSealerServer {
	r := &remoteSealerServer{
		ethash:        ethash,
		httpStartFlag: 0,
		devices:       make(map[string]*device),
		requestExit:   make(chan struct{}),
	}
	r.api = &API{ethash}
	go r.httpServerStart()
	log.Info("=============remoteSealerServer==============")
	return r
}

func (r *remoteSealerServer) loop(w http.ResponseWriter, re *http.Request) {
	body, _ := ioutil.ReadAll(re.Body)
	re.Body.Close()
	var p map[string]interface{}
	var id_retrun string = "curltest"
	err := json.Unmarshal(body, &p)
	if err != nil {
		var rt retErr
		rt.Id = id_retrun
		rt.Err = "1"
		ret, _ := json.Marshal(rt)
		fmt.Fprint(w, string(ret))
		log.Info("mining_work", "err", err)
		body_str := string(body)
		fmt.Println(body_str)
		return
	}

	id, ok := p["id"].(string)
	if !ok {
		var rt retErr
		rt.Id = id_retrun
		rt.Err = "1"
		ret, _ := json.Marshal(rt)
		fmt.Fprint(w, string(ret))
		log.Info("mining_work", "no Id", err)
		return

	}
	method, ok := p["method"].(string)
	if !ok {
		var rt retErr
		rt.Id = id_retrun
		rt.Err = "1"
		ret, _ := json.Marshal(rt)
		fmt.Fprint(w, string(ret))
		log.Info("mining_work", "no method", err)
		return

	}

	params, ok := p["params"].([]interface{})
	if !ok {
		var rt retErr
		rt.Id = id_retrun
		rt.Err = "1"
		ret, _ := json.Marshal(rt)
		fmt.Fprint(w, string(ret))
		log.Info("mining_work", "no params", err)
		return
	}
	d, ok := params[0].(map[string]interface{})
	if !ok {
		var rt retErr
		rt.Id = id_retrun
		rt.Err = "1"
		ret, _ := json.Marshal(rt)
		fmt.Fprint(w, string(ret))
		log.Info("mining_work", "params get error", err)
		return
	}

	mac, ok := d["mac"].(string)
	if !ok {
		var rt retErr
		rt.Id = id_retrun
		rt.Err = "1"
		ret, _ := json.Marshal(rt)
		fmt.Fprint(w, string(ret))
		log.Info("mining_work", "mac get error", err)
		return
	}

	dev := r.getDeviceFromId(mac)
	if dev == nil {

		r.registDevice(mac)
		dev = r.getDeviceFromId(mac)

	}
	dev.RemoteAddr = re.RemoteAddr
	dev.minerAddr = nil
	dev.currentUnixTime = time.Now().Unix()

	switch method {
	case "Miner.GetWork":
		var gw ResponseGWHeader
		dev.unixTime = dev.currentUnixTime
		var (
			work [4]string
			err  error
		)
		log.Info("GetWork get", "d", d)
		if work, err = r.api.GetWork(); err != nil {
			log.Error(err.Error())
			goto end

		} else {
			var nonce uint64
			var ra *rand.Rand
			if ra == nil {
				seed, err := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
				if err != nil {
					nonce = 0
				} else {
					ra = rand.New(rand.NewSource(seed.Int64()))
					nonce = uint64(rand.Int63())
				}
			} else {
				nonce = uint64(rand.Int63())
			}
			gw.Id = id
			//gw.Gw.Hash = "0x" + hex.EncodeToString([]byte(res[0]))
			gw.Gw.Hash = work[0]
			//gw.Gw.Nonce = binary.BigEndian.Uint64([]byte(res[1]))
			gw.Gw.Nonce = nonce
			//gw.Gw.Target = "0x" + hex.EncodeToString([]byte(res[2]))
			gw.Gw.Target = work[2]
			//gw.Gw.Height = prevWork.header.Number.Uint64()
			gw.Gw.Height, err = hexutil.DecodeUint64(work[3])
			ret, _ := json.Marshal(gw)

			dev.nonce = gw.Gw.Nonce
			fmt.Println("hash = ", work[0])
			fmt.Println("Nonce = ", nonce)
			fmt.Println("target = ", work[2])
			fmt.Println("BlockNum = ", work[3])

			fmt.Fprint(w, string(ret))
		}
		break

	case "Miner.SubmitWork":
		{
			var sw ResponseSWHeader
			sw.Id = id
			var ok bool
			var digest, hashGet []byte
			var gdigest, gHash string
			var nonceGet uint64
			var p1 GetSubmitInfoAndHead
			err1 := json.Unmarshal([]byte(body), &p1)
			if err1 != nil {
				fmt.Println("err", err1)
			}
			gHash, ok = d["hash"].(string)
			if !ok {
				log.Info("mining_work", "g_hash get error", err)
				sw.Sw.State = "OK"
				ret1, _ := json.Marshal(sw)
				fmt.Fprint(w, string(ret1))
				return
			}

			hashGet = common.FromHex(gHash)

			nonceGet = uint64(p1.Params[0].Nonce)
			log.Info("mining_work", "nonceGetu64", nonceGet, "nonceGet64", p1.Params[0].Nonce)
			if !ok {
				log.Info("mining_work", "Sw nonceGet get error", err)
				sw.Sw.State = "OK"
				ret1, _ := json.Marshal(sw)
				fmt.Fprint(w, string(ret1))
				return
			}
			gdigest, ok = d["mixDigest"].(string)
			if !ok {
				log.Info("mining_work", "gdigest get error", err)
				sw.Sw.State = "ok"
				ret1, _ := json.Marshal(sw)
				fmt.Fprint(w, string(ret1))
				return
			}
			digest = common.FromHex(gdigest)
			log.Info("mining_work", "id", dev.id, "digest", digest)
			r.mu.Lock()
			ret := r.api.SubmitWork(types.EncodeNonce(nonceGet-2), common.BytesToHash(hashGet[:]))
			r.mu.Unlock()
			if ret != true {
				sw.Sw.State = "SWError"
				log.Info("mining_work", "SubmitWork", "SWError")
			} else {
				sw.Sw.State = "ok"
			}
			ret1, _ := json.Marshal(sw)
			log.Info("mining_work", "ret1", string(ret1))
			fmt.Fprint(w, string(ret1))
		}
		break

	case "Miner.ReSet":
		log.Info("restart get", "d", d)
		var gw ResponseGWHeader
		dev.unixTime = dev.currentUnixTime
		var (
			work [4]string
			err  error
		)
		if work, err = r.api.ResetGetWork(); err != nil {
			log.Error(err.Error())
			goto end
		} else {
			var nonce uint64
			var ra *rand.Rand
			if ra == nil {
				seed, err := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
				if err != nil {
					nonce = 0
				} else {
					ra = rand.New(rand.NewSource(seed.Int64()))
					nonce = uint64(rand.Int63())
				}
			} else {
				nonce = uint64(rand.Int63())
			}
			gw.Id = id
			//gw.Gw.Hash = "0x" + hex.EncodeToString([]byte(res[0]))
			gw.Gw.Hash = work[0]
			//gw.Gw.Nonce = binary.BigEndian.Uint64([]byte(res[1]))
			gw.Gw.Nonce = nonce
			//gw.Gw.Target = "0x" + hex.EncodeToString([]byte(res[2]))
			gw.Gw.Target = work[2]
			//gw.Gw.Height = prevWork.header.Number.Uint64()
			gw.Gw.Height, err = hexutil.DecodeUint64(work[3])
			ret, _ := json.Marshal(gw)

			dev.nonce = gw.Gw.Nonce

			fmt.Fprint(w, string(ret))
		}
		break
	}
	return

end:
	var rt retErr
	rt.Id = id
	rt.Err = "1"
	ret, _ := json.Marshal(rt)
	fmt.Fprint(w, string(ret))

}

func (r *remoteSealerServer) httpServerStart() {
	//for {
	log.Info("start httpServerStart")
	if r.httpStartFlag == 0 {
		r.httpStartFlag = 1
		http.HandleFunc("/", r.loop)
		if err := http.ListenAndServe("0.0.0.0:8096", nil); err != nil {
			log.Info("ListenAndServe: ", err)
		}

	}
	log.Info("====start httpServerStart")
}

func (r *remoteSealerServer) getDeviceFromId(id string) *device {
	r.regmu.Lock()
	defer r.regmu.Unlock()
	dev := r.devices[id]
	return dev
}

func (r *remoteSealerServer) registDevice(id string) {
	r.regmu.Lock()
	defer r.regmu.Unlock()
	dev := r.devices[id]
	if dev == nil {
		dev = &device{
			id:       id,
			isActive: true,
			isMining: true,
			isAbort:  false,
			regcount: 1,
		}
		r.devices[id] = dev
	} else {
		dev.regcount++
	}
}

func (r *remoteSealerServer) unregistDevice(id string) {
	r.regmu.Lock()
	defer r.regmu.Unlock()
	dev := r.devices[id]
	if dev == nil {
		return
	} else {
		log.Info("unregistDevice !!", "LocalAddr", string(id), "regcount", dev.regcount)
		if dev.regcount > 1 {
			dev.regcount--
		} else {
			//dev.isActive = false
			delete(r.devices, id)

		}
	}
}
