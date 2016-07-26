package teegon

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shopex/teegon-open-go-sdk/message"
)

func (me *Client) Notify() (n *Notify, err error) {
	n = &Notify{Client: me}
	err = n.dail()
	return
}

const (
	command_publish byte = 1
	command_consume byte = 2
	command_ack     byte = 3
)

type Notify struct {
	Client     *Client
	conn       *websocket.Conn
	hbChan     chan bool
	subSuccess bool
}

type Delivery struct {
	App_Key string
	Message *message.MsgData
	Time    int64
	notify  *Notify
}

func (d *Delivery) Ack() error {

	ack := message.Command{
		Cmd:       message.MessageAckCmd,
		RequestId: GetRequestId(),
		Body: message.MsgAck{
			Group:     d.Message.Group,
			Topic:     d.Message.Topic,
			Msgid:     d.Message.Offset,
			Partition: d.Message.Partition,
		}}
	if data, err := d.notify.packProtocol(ack); err == nil {
		return d.notify.conn.WriteMessage(1, data)
	} else {
		return err
	}

}

//func (d *message.Response) Ack() error {
//	buf := bytes.NewBuffer([]byte{command_ack})
//	buf.WriteString(strconv.FormatInt(d.Tag, 10))
//	return d.conn.WriteMessage(1, buf.Bytes())
//}

func (n *Notify) retry() <-chan bool {
	ch := make(chan bool)
	if n.Client == nil {
		log.Printf("can not retry on a nil cilent")
		return ch
	}

	if n.conn != nil {
		n.Close()
	}

	go func() {
		// 30秒重连
		c := time.Tick(time.Second * 30)

		var err error
		for {
			<-c
			err = n.dail()
			if err != nil {
				log.Printf("reconnect to websocket fail (%s)\n", err)
				continue
			}
			ch <- true
			return
		}
	}()
	return ch
}

func (n *Notify) dail() (err error) {
	req, err := n.Client.getRequest("GET", "platform/notify", nil)
	tcpcon, err := net.Dial("tcp", req.URL.Host)
	if err != nil {
		return err
	}
	req.URL.Scheme = "ws"
	n.conn, _, err = websocket.NewClient(tcpcon, req.URL, req.Header, 128, 128)
	if err == nil { //开启心跳
		n.hbChan = n.runHeartBeat()
	}
	return
}

func (n *Notify) consume(topic string, prefetch int, ch chan *Delivery) error {
	if n.subSuccess {
		return errors.New("has subscribed other queue.")
	}

	if topic == "" {
		return errors.New("topic is empty.")
	}
	cmd := message.Command{
		Cmd: message.SubMessageCmd,
		Body: message.SubMessage{
			Topic: topic,
		},
	}
	data, perr := n.packProtocol(cmd)
	if perr != nil {
		return perr
	}

	err := n.conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		log.Printf("write to websocket err (%s)\n", err)
		return err
	}

	go func() { //收发数据
		defer func() {
			if err := recover(); err != nil {
				log.Printf("(*Notify).Consume meet a panic (%s)\n", err)
				ok := n.retry()
				<-ok
				n.consume(topic, prefetch, ch)
			}
		}()
		for {
			_, data, err := n.conn.ReadMessage()
			if err != nil {
				log.Printf("conn read error:%s", err.Error())
				panic(err)
			}

			rep := &message.Response{}
			err = json.Unmarshal(data, rep)
			log.Printf("read msg sdsd:%v, sd:%v", err, rep)
			//d.conn = n.conn
			if err == nil {
				if rep.Command == message.MessageNotifyCmd {
					d := &Delivery{
						App_Key: n.Client.Key,
						Message: rep.Result.(*message.MsgData),
						Time:    time.Now().Unix(),
						notify:  n,
					}

					ch <- d
				}
			}
		}
	}()

	n.subSuccess = true

	return nil
}

func (n *Notify) runHeartBeat() chan bool {
	ch := make(chan bool)
	go func() {
		c := time.Tick(time.Second * 8) //3秒一次心跳
		for {
			select {
			case <-c:
				n.sendHeartBeat()
			case <-ch:
				return
			}
		}
	}()
	return ch
}

func (n *Notify) Consume(topic string) (ch chan *Delivery, err error) {
	ch = make(chan *Delivery)
	err = n.consume(topic, 1, ch)
	return ch, err
}

func (n *Notify) encode(v interface{}) (bin []byte) {
	switch v.(type) {
	case []byte:
		bin = v.([]byte)
	case string:
		bin = []byte(v.(string))
	default:
		bin, _ = json.Marshal(v)
	}
	return
}

func (n *Notify) sendHeartBeat() error {
	cmd := message.Command{
		Cmd:       message.HeartBeatCmd,
		RequestId: GetRequestId(),
	}
	if data, err := n.packProtocol(cmd); err == nil {
		err = n.conn.WriteMessage(websocket.TextMessage, data)
		return err
	} else {
		return err
	}
}

func (n *Notify) packProtocol(cmd message.Command) ([]byte, error) {
	vals := url.Values{}
	vals.Set("method", cmd.Cmd)
	vals.Set("app_key", n.Client.Key)

	vals.Set("sign_time", strconv.FormatInt(time.Now().Unix(), 10))

	strCmd, err := json.Marshal(cmd)
	if err != nil {
		log.Printf("marshal command err (%s)\n", err)
		return nil, err
	}
	//fmt.Println("cmd:%s", string(strCmd))
	vals.Set("body", base64.URLEncoding.EncodeToString(strCmd))

	req := &http.Request{URL: &url.URL{}}
	req.URL.RawQuery = vals.Encode()
	vals.Set("sign", Sign(req, n.Client.secret))
	data := []byte(vals.Encode())

	return data, nil
}

func (n *Notify) Pub(topic string, key string, body string) (err error) {
	cmd := message.Command{
		Cmd:       message.WriteMessageCmd,
		RequestId: GetRequestId(),
		Body: message.WriteMessage{
			Topic: topic,
			Key:   key,
			Data:  body,
		},
	}

	if data, err := n.packProtocol(cmd); err == nil {
		err = n.conn.WriteMessage(1, data)
		return err
	} else {
		return err
	}
}

func (n *Notify) Close() error {
	n.hbChan <- true
	n.subSuccess = false
	return n.conn.Close()
}
