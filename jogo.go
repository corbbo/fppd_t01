package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/nsf/termbox-go"
)

var mutex sync.Mutex

// Define os elementos do jogo
type Elemento struct {
	simbolo     rune
	cor         termbox.Attribute
	corFundo    termbox.Attribute
	tangivel    bool
	canBeKilled bool
}

// Personagem controlado pelo jogador
var personagem = Elemento{
	simbolo:     '☺',
	cor:         termbox.ColorWhite,
	corFundo:    termbox.ColorDefault,
	tangivel:    true,
	canBeKilled: true,
}

// Parede
var parede = Elemento{
	simbolo:     '▣',
	cor:         termbox.ColorBlack | termbox.AttrBold | termbox.AttrDim,
	corFundo:    termbox.ColorDarkGray,
	tangivel:    true,
	canBeKilled: false,
}

// Barrreira
var barreira = Elemento{
	simbolo:     '#',
	cor:         termbox.ColorRed,
	corFundo:    termbox.ColorDefault,
	tangivel:    true,
	canBeKilled: false,
}

// Vegetação
var vegetacao = Elemento{
	simbolo:     '♣',
	cor:         termbox.ColorGreen,
	corFundo:    termbox.ColorDefault,
	tangivel:    false,
	canBeKilled: false,
}

// Elemento vazio
var vazio = Elemento{
	simbolo:     ' ',
	cor:         termbox.ColorDefault,
	corFundo:    termbox.ColorDefault,
	tangivel:    false,
	canBeKilled: false,
}

// Elemento para representar áreas não reveladas (efeito de neblina)
var neblina = Elemento{
	simbolo:     '.',
	cor:         termbox.ColorDefault,
	corFundo:    termbox.ColorYellow,
	tangivel:    false,
	canBeKilled: false,
}
var inimigo = Elemento{
	simbolo:     '☠',
	cor:         termbox.ColorRed,
	corFundo:    termbox.ColorDefault,
	tangivel:    true,
	canBeKilled: false,
}
var chave = Elemento{
	simbolo:     '⚷',
	cor:         termbox.ColorYellow,
	corFundo:    termbox.ColorDefault,
	tangivel:    false,
	canBeKilled: false,
}
var porta = Elemento{
	simbolo:     '⚑',
	cor:         termbox.ColorYellow,
	corFundo:    termbox.ColorBlack,
	tangivel:    true,
	canBeKilled: false,
}
var npc = Elemento{
	simbolo:     'ﭶ',
	cor:         termbox.ColorBlue,
	corFundo:    termbox.ColorDefault,
	tangivel:    true,
	canBeKilled: true,
}

var portal = Elemento{
	simbolo:     'ೱ',
	cor:         termbox.ColorCyan,
	corFundo:    termbox.ColorBlack,
	tangivel:    false,
	canBeKilled: false,
}

type Enemy struct {
	x, y  int
	elem  Elemento
	alive bool
}
type NonPlayerChar struct {
	x, y        int
	elem        Elemento
	interacted  bool
	canBeKilled bool
}

type _portal struct {
	x, y       int
	elem       Elemento
	interacted bool
}

var interable = [4]rune{chave.simbolo, porta.simbolo, npc.simbolo, portal.simbolo}
var functions = [4]func(x int, y int){interact_chave, interact_porta, interact_npc, interact_portal}

var i = Enemy{x: 0, y: 0, elem: inimigo, alive: true}
var n = NonPlayerChar{x: 0, y: 0, elem: npc, interacted: false, canBeKilled: true}
var p = _portal{x: 0, y: 0, elem: portal, interacted: false}
var winnable = false
var ded = false
var borked = false

var mapa [][]Elemento
var posX, posY int
var ultimoElementoSobPersonagem = vazio
var statusMsg string
var ganhei = false
var doubleSPEED = false
var passos = 0

var doneNPC = make(chan bool)
var doneInimigo = make(chan bool)

var efeitoNeblina = false
var revelado [][]bool
var raioVisao int = 3

func main() {
	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	defer termbox.Close()

	carregarMapa("mapa.txt")
	if efeitoNeblina {
		revelarArea()
	}

	desenhaTudo()
	go logicaInimigoLui()
	go logicNPC()
	go checkEnemy2cell()
	go checkEnemy1cell()
	go logicaPortal()

	for !(ganhei || ded) {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			if ev.Key == termbox.KeyEsc {
				return // Sair do programa
			}
			if ev.Ch == 'e' {
				interagir()
			} else {
				mover(ev.Ch)
				if efeitoNeblina {
					revelarArea()
				}
			}
			desenhaTudo()
		}
	}
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	if ded == true {
		fmt.Println("Você perdeu!")
	} else {
		fmt.Println("Você ganhou!")
	}
	time.Sleep(2 * time.Second)
}

func carregarMapa(nomeArquivo string) {
	arquivo, err := os.Open(nomeArquivo)
	if err != nil {
		panic(err)
	}
	defer arquivo.Close()

	scanner := bufio.NewScanner(arquivo)
	y := 0
	for scanner.Scan() {
		linhaTexto := scanner.Text()
		var linhaElementos []Elemento
		var linhaRevelada []bool
		for x, char := range linhaTexto {
			elementoAtual := vazio
			switch char {
			case parede.simbolo:
				elementoAtual = parede
			case barreira.simbolo:
				elementoAtual = barreira
			case vegetacao.simbolo:
				elementoAtual = vegetacao
			case personagem.simbolo:
				// Atualiza a posição inicial do personagem
				posX, posY = x, y
				elementoAtual = vazio
			case inimigo.simbolo:
				i.x, i.y = x, y
				elementoAtual = vazio
			case chave.simbolo:
				elementoAtual = chave
			case porta.simbolo:
				elementoAtual = porta
			case npc.simbolo:
				n.x, n.y = x, y
				elementoAtual = vazio
			}
			linhaElementos = append(linhaElementos, elementoAtual)
			linhaRevelada = append(linhaRevelada, false)
		}
		mapa = append(mapa, linhaElementos)
		revelado = append(revelado, linhaRevelada)
		y++
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
}

func desenhaTudo() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	for y, linha := range mapa {
		for x, elem := range linha {
			if efeitoNeblina == false || revelado[y][x] {
				termbox.SetCell(x, y, elem.simbolo, elem.cor, elem.corFundo)
			} else {
				termbox.SetCell(x, y, neblina.simbolo, neblina.cor, neblina.corFundo)
			}
		}
	}

	desenhaBarraDeStatus()

	termbox.Flush()
}

func desenhaBarraDeStatus() {
	for i, c := range statusMsg {
		termbox.SetCell(i, len(mapa)+1, c, termbox.ColorCyan, termbox.ColorDefault)
	}
	msg := "Use WASD para mover e E para interagir. ESC para sair."
	for i, c := range msg {
		termbox.SetCell(i, len(mapa)+3, c, termbox.ColorCyan, termbox.ColorDefault)
	}
}

func revelarArea() {
	minX := max(0, posX-raioVisao)
	maxX := min(len(mapa[0])-1, posX+raioVisao)
	minY := max(0, posY-raioVisao/2)
	maxY := min(len(mapa)-1, posY+raioVisao/2)

	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			// Revela as células dentro do quadrado de visão
			revelado[y][x] = true
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func mover(comando rune) {
	dx, dy := 0, 0
	switch comando {
	case 'w':
		if doubleSPEED {
			dy = -2
			passos = passos + 2
		} else {
			dy = -1
		}
	case 'a':
		if doubleSPEED {
			dx = -2
			passos = passos + 2
		} else {
			dx = -1
		}
	case 's':
		if doubleSPEED {
			dy = 2
			passos = passos + 2
		} else {
			dy = 1
		}
	case 'd':
		if doubleSPEED {
			dx = 2
			passos = passos + 2
		} else {
			dx = 1
		}
	}
	novaPosX, novaPosY := posX+dx, posY+dy
	if novaPosY >= 0 && novaPosY < len(mapa) && novaPosX >= 0 && novaPosX < len(mapa[novaPosY]) &&
		mapa[novaPosY][novaPosX].tangivel == false {
		mapa[posY][posX] = ultimoElementoSobPersonagem         // Restaura o elemento anterior
		ultimoElementoSobPersonagem = mapa[novaPosY][novaPosX] // Atualiza o elemento sob o personagem
		posX, posY = novaPosX, novaPosY                        // Move o personagem
		mapa[posY][posX] = personagem                          // Coloca o personagem na nova posição
	}
}

func interagir() {
	//para cada celula na matriz num raio de 2 celulas, interage com o elemento mais próximo
	for y := max(0, posY-2); y <= min(len(mapa)-1, posY+2); y++ {
		for x := max(0, posX-2); x <= min(len(mapa[y])-1, posX+2); x++ {
			for count := 0; count < len(interable); count++ {
				if mapa[y][x].simbolo == interable[count] {
					functions[count](x, y)
				}
			}
		}
	}
}

func logicaInimigoLui() {
	for !(ganhei || ded) {
		// select {
		// case <-doneInimigo:
		// 	return
		// default:
		curX, curY := i.x, i.y
		speedX := rand.Intn(3) - 1 // Generate a random speed for X direction (-1, 0, 1)
		speedY := rand.Intn(3) - 1 // Generate a random speed for Y direction (-1, 0, 1)

		var novaPosX, novaPosY int

		for {
			novaPosX, novaPosY = curX+speedX, curY+speedY
			if novaPosY >= 0 && novaPosY < len(mapa) && novaPosX >= 0 && novaPosX < len(mapa[novaPosY]) &&
				(mapa[novaPosY][novaPosX].tangivel && !mapa[novaPosY][novaPosX].canBeKilled) == false {
				i.x += speedX // Update i's X position
				i.y += speedY // Update i's Y position
				break
			} else {
				speedX = rand.Intn(3) - 1 // Generate a random speed for X direction (-1, 0, 1)
				speedY = rand.Intn(3) - 1 // Generate a random speed for Y direction (-1, 0, 1)
			}
		}
		mutex.Lock()
		mapa[curY][curX] = vazio // Clear previous i position on the map
		mapa[i.y][i.x] = inimigo // Update i's new position on the map
		desenhaTudo()
		mutex.Unlock()
		if borked {
			mutex.Lock()
			statusMsg = "O Inimigo levou stun!"
			mutex.Unlock()
			time.Sleep(800 * time.Millisecond) // Pause for a short duration
		} else {
			time.Sleep(100 * time.Millisecond) // Pause for a short duration
		}

	}
}

//}

func logicNPC() {

	for !(ganhei || ded) {
		select {
		case <-doneNPC:
			return
		default:

			curX, curY := n.x, n.y
			initX, initY := n.x, n.y

			speedX := rand.Intn(3) - 1 // Generate a random speed for X direction (-1, 0, 1)
			speedY := rand.Intn(3) - 1 // Generate a random speed for Y direction (-1, 0, 1)
			var novaPosX, novaPosY int

			for {
				novaPosX, novaPosY = curX+speedX, curY+speedY
				if novaPosY >= 0 && novaPosY < len(mapa) && novaPosX >= 0 && novaPosX < len(mapa[novaPosY]) &&
					mapa[novaPosY][novaPosX].tangivel == false {
					n.x += speedX // Update n's X position
					n.y += speedY // Update n's Y position
					break
				} else {
					speedX = rand.Intn(3) - 1 // Generate a random speed for X direction (-1, 0, 1)
					speedY = rand.Intn(3) - 1 // Generate a random speed for Y direction (-1, 0, 1)
				}
			}
			mutex.Lock()
			mapa[initY][initX] = vazio // Clear previous i position on the map
			mapa[n.y][n.x] = npc       // Update i's new position on the map
			desenhaTudo()
			mutex.Unlock()
			time.Sleep(3000 * time.Millisecond) // Pause for a short duration
		}
	}
}

func passosDados() {
	for passos < 40 {
	}
	doubleSPEED = false
	passos = 0
	go logicNPC()
	return
}
func checkEnemy2cell() /* pode matar chave e npc*/ {
	//para cada celula na matriz num raio de 2 celulas, interage com o elemento mais próximo

	for !(ganhei || ded) {

		for y := max(0, i.y-2); y <= min(len(mapa)-1, i.y+2); y++ {
			for x := max(0, i.x-2); x <= min(len(mapa[y])-1, i.x+2); x++ {
				if mapa[y][x].simbolo == chave.simbolo {
					winnable = true

					mutex.Lock()
					statusMsg = "O inimigo matou a chave, corra para a porta!"
					mapa[y][x] = vazio
					desenhaTudo()
					borked = true
					mutex.Unlock()

				} else if mapa[y][x].simbolo == npc.simbolo {
					if n.canBeKilled {
						mutex.Lock()
						statusMsg = "O inimigo matou o NPC!"

						n.elem = vazio
						mapa[y][x] = vazio
						desenhaTudo()
						borked = true
						mutex.Unlock()

						doneNPC <- true

					} else {
						mutex.Lock()
						statusMsg = "O inimigo não pode matar o NPC!"
						mutex.Unlock()
					}
				}
				if borked {
					time.Sleep(500 * time.Millisecond) // Pause for a short duration
					mutex.Lock()
					borked = false
					mutex.Unlock()
				}
			}
		}

	}
}
func checkEnemy1cell() /* pode matar parede e jogador*/ {
	for !(ganhei || ded) {
		//para cada celula na matriz num raio de 1 celulas, interage com o elemento mais próximo
		for y := max(0, i.y-1); y <= min(len(mapa)-1, i.y+1); y++ {
			for x := max(0, i.x-1); x <= min(len(mapa[y])-1, i.x+1); x++ {
				if mapa[y][x].simbolo == parede.simbolo {
					mutex.Lock()
					statusMsg = "O inimigo quebrou uma parede!"
					mapa[y][x] = vazio
					desenhaTudo()
					borked = true
					mutex.Unlock()

				} else if mapa[y][x].simbolo == personagem.simbolo {
					if personagem.canBeKilled {
						mutex.Lock()
						statusMsg = "O inimigo te matou!"
						n.elem = vazio
						mapa[y][x] = vazio
						desenhaTudo()
						mutex.Unlock()
						ded = true
					} else {
						mutex.Lock()
						statusMsg = "O inimigo não pode te matar!"
						mutex.Unlock()
					}
				}
				if borked {
					mutex.Lock()
					time.Sleep(1000 * time.Millisecond) // Pause for a short duration
					borked = false                      //só pode quebrar a parede uma vez a cada segundo
					mutex.Unlock()
				}
			}
		}

	}
}
func logicaPortal() {
	for !(ganhei || ded) {
		time.Sleep(time.Duration(rand.Intn(7)) * time.Second)
		p.y = posY + rand.Intn(3) - 1
		p.x = posX + rand.Intn(3) - 1
		if p.y == posY && p.x == posX {
			p.y++
			p.x++
		}
		mutex.Lock()
		mapa[p.y][p.x] = portal
		desenhaTudo()
		mutex.Unlock()
		time.Sleep(5 * time.Second)
		mutex.Lock()
		mapa[p.y][p.x] = vazio
		desenhaTudo()
		mutex.Unlock()
		time.Sleep(5 * time.Second)
	}
}

func interact_chave(x int, y int) {
	statusMsg = "Você pegou a chave!"
	winnable = true
	mutex.Lock()
	mapa[y][x] = vazio
	mutex.Unlock()
}
func interact_porta(x int, y int) {
	if winnable { //only lets player win if he has the key
		statusMsg = "Você abriu a porta! Parabéns!"
		mutex.Lock()
		mapa[y][x] = vazio
		mutex.Unlock()
		ganhei = true
	} else {
		statusMsg = "Você precisa da chave!"
	}
}
func interact_npc(x int, y int) {
	if !n.interacted {
		statusMsg = "Você ganhou o buff de velocidade! (40 passos rápidos)"
		doubleSPEED = true
		go passosDados()
		n.elem = vazio
		mutex.Lock()
		mapa[y][x] = vazio
		mutex.Unlock()
		doneNPC <- true
		n.interacted = true
	} else {
		statusMsg = "Você já interagiu com o NPC!"
	}
}
func interact_portal(x int, y int) {
	var teleX int
	var teleY int
	mutex.Lock()
	mapa[posY][posX] = vazio
	mapa[y][x] = vazio
	mutex.Unlock()

	desenhaTudo()
	teleX = (rand.Intn(90) - rand.Intn(90))
	teleY = (rand.Intn(90) - rand.Intn(90))
	if teleX >= len(mapa[y])-1 {
		teleX = len(mapa[y]) - 2
	}
	if teleX <= 0 {
		teleX = 2
	}
	if teleY <= 0 {
		teleY = 2
	}
	if teleY >= len(mapa)-1 {
		teleY = len(mapa) - 2
	}
	mutex.Lock()
	posX = teleX //isso possivelmente tá dando o erro do mapa quebrar quando o personagem entra no portal
	posY = teleY
	mutex.Unlock()
	desenhaTudo()
}
