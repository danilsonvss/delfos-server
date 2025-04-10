package main

import (
	"fmt"
	"log"
	"net"
	"sync"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/gordonklaus/portaudio"
)

var (
	buffer       = make([]int16, 512)
	stream       *portaudio.Stream
	volumeGain   = 2.0
	conn         net.Conn
	stopStream   = make(chan bool)
	selectedIP   = ""
	selectedDev  *portaudio.DeviceInfo
	started      = false
	streamLocker sync.Mutex
	startBtn 	 *widget.Button
)

func main() {
	portaudio.Initialize()
	defer portaudio.Terminate()

	myApp := app.New()
	myWindow := myApp.NewWindow("Delfos")
	myWindow.SetTitle("Delfos - Transmissão de Áudio")
	myWindow.SetIcon(nil)
	myWindow.SetFixedSize(true)
	myWindow.Resize(fyne.NewSize(400, 300))

	// Interfaces de áudio
	devices, _ := portaudio.Devices()
	var deviceNames []string
	deviceMap := map[string]*portaudio.DeviceInfo{}
	for _, d := range devices {
		if d.MaxInputChannels > 0 {
			deviceNames = append(deviceNames, d.Name)
			deviceMap[d.Name] = d
		}
	}
	deviceSelect := widget.NewSelect(deviceNames, func(s string) {
		selectedDev = deviceMap[s]
	})

	// Slider de ganho
	gainSlider := widget.NewSlider(1, 12)
	gainSlider.Step = 0.1
	gainSlider.Value = 1.0
	gainLabel := widget.NewLabel("Ganho: 1.0")
	gainSlider.OnChanged = func(val float64) {
		volumeGain = val
		gainLabel.SetText("Ganho: " + fmt.Sprintf("%.1f", val))
	}

	// Lista de IPs detectados
	ipSelect := widget.NewSelect([]string{}, func(s string) {
		selectedIP = s
	})
	ipSelect.PlaceHolder = "Aguardando dispositivos..."
	ipSelect.Refresh()
	listenPresence(ipSelect)

	// Botão iniciar/parar
	startBtn = widget.NewButton("Iniciar Transmissão", func() {
		if !started {
			go startStreaming()
			started = true
			startBtn.SetText("Parar")
		} else {
			stopStream <- true
			started = false
			startBtn.SetText("Iniciar Transmissão")
		}
	})

	myWindow.SetContent(container.NewVBox(
		widget.NewLabel("Interface de Áudio:"),
		deviceSelect,
		gainLabel,
		gainSlider,
		widget.NewLabel("IP do Receptor:"),
		ipSelect,
		startBtn,
	))

	myWindow.ShowAndRun()
}

func listenPresence(ipSelect *widget.Select) {
	pc, err := net.ListenPacket("udp", ":9999")
	if err != nil {
		log.Println("Erro ao escutar porta 9999:", err)
		return
	}
	go func() {
		defer pc.Close()
		buf := make([]byte, 1024)
		for {
			n, _, err := pc.ReadFrom(buf)
			if err != nil {
				continue
			}
			msg := string(buf[:n])
			if strings.HasPrefix(msg, "DELFOS_ONLINE:") {
				senderIP := strings.TrimPrefix(msg, "DELFOS_ONLINE:")
				log.Println("IP", senderIP)

				found := false
				for _, existing := range ipSelect.Options {
					if existing == senderIP {
						found = true
						break
					}
				}
				if !found {
					options := append(ipSelect.Options, senderIP)
					ipSelect.SetOptions(options)
				}
			}
		}
	}()
}

func startStreaming() {
	if selectedDev == nil || selectedIP == "" {
		log.Println("Selecione interface de áudio e IP.")
		return
	}

	streamLocker.Lock()
	defer streamLocker.Unlock()

	var err error
	stream, err = portaudio.OpenStream(portaudio.StreamParameters{
		Input: portaudio.StreamDeviceParameters{
			Device:   selectedDev,
			Channels: 2,
			Latency:  selectedDev.DefaultLowInputLatency,
		},
		SampleRate:      44100,
		FramesPerBuffer: len(buffer),
	}, buffer)
	if err != nil {
		log.Println("Erro ao abrir stream:", err)
		return
	}

	conn, err = net.Dial("udp", selectedIP+":9999")
	if err != nil {
		log.Println("Erro na conexão UDP:", err)
		stream.Close() // fecha se abrir, mesmo que erro seja no conn
		return
	}

	err = stream.Start()
	if err != nil {
		log.Println("Erro ao iniciar stream:", err)
		conn.Close()
		stream.Close()
		return
	}

	log.Println("Transmitindo áudio para", selectedIP)

	go func() {
		defer func() {
			if err := stream.Stop(); err != nil {
				log.Println("Erro ao parar stream:", err)
			}
			if err := stream.Close(); err != nil {
				log.Println("Erro ao fechar stream:", err)
			}
			conn.Close()
			log.Println("Transmissão encerrada.")
		}()

		for {
			select {
			case <-stopStream:
				log.Println("Parando transmissão.")
				return
			default:
				if err := stream.Read(); err != nil {
					log.Println("Erro ao ler:", err)
					continue
				}
				adjusted := applyVolumeGain(buffer, volumeGain)
				conn.Write(int16ToBytes(adjusted))
			}
		}
	}()
}

func applyVolumeGain(samples []int16, gain float64) []int16 {
	adjusted := make([]int16, len(samples))
	for i, s := range samples {
		val := float64(s) * gain
		if val > 32767 {
			val = 32767
		} else if val < -32768 {
			val = -32768
		}
		adjusted[i] = int16(val)
	}
	return adjusted
}

func int16ToBytes(samples []int16) []byte {
	buf := make([]byte, len(samples)*2)
	for i, s := range samples {
		buf[2*i] = byte(s)
		buf[2*i+1] = byte(s >> 8)
	}
	return buf
}
