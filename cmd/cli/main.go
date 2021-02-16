package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cofepy/gn/cmd/util"
)

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

var headerLen = 2

func waitServerResponse(conn net.Conn) {
	// 循环读取server返回的数据
	for {
		// 创建解码器
		codec := util.NewCodec(conn)
		// log.Println("new loop")
		// 从解码器中读数据
		_, err := codec.Read()

		if err != nil {
			log.Println("解码异常，退出程序, error:", err)
			conn.Close()
			os.Exit(0)
			return
		}
		bytes, ok, err := codec.Decode()

		// 解码出错，需要中断连接
		if err != nil {
			log.Println(err)
			return
		}
		// 数据读取正常
		if ok {
			log.Println("收到数据【", string(bytes), "】")
			continue
		} else {
			log.Println("空数据")
		}
		// time.Sleep(time.Second * 3)
	}
}

func sendServerData(conn net.Conn) {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("要给服务端发送数据，请按enter键，输入q退出：")

		for {
			time.Sleep(time.Second * 1)
			text, _ := reader.ReadString('\n')

			if strings.Compare(text, "\n") == 0 {
				fmt.Println("开始发送数据")
				break

			} else {
				// convert CRLF to LF
				text = strings.Replace(text, "\n", "", -1)
				if strings.Compare(text, "q") == 0 {
					fmt.Println("退出程序")
					conn.Close()
					os.Exit(0)
				}
			}

		}

		for i := 0; i < 3; i++ {
			str := "hello" + strconv.Itoa(i)
			data := util.Encode([]byte(str))
			fmt.Println("发送数据:【", str, "】")
			_, err := conn.Write(data)
			if err != nil {
				log.Println(err)
				return
			}
		}
	}
}

func main() {

	log.Println("start")

	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		log.Println("Error dialing", err.Error())
		return // 终止程序
	}

	go waitServerResponse(conn)

	go sendServerData(conn)

	select {}

}
