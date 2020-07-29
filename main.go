package main

import (
	"fmt"
	"sync"
	"time"
)

const ( //estados del barbero que el cliente verificarà
	durmiendo = iota
	observando
	cortando
)

var stateLog = map[int]string{
	0: "Durmiendo",
	1: "Observando",
	2: "Cortando",
}
var wg *sync.WaitGroup // para la cantidad tope de clientes el tope es 5

type Barbero struct {
	name string
	sync.Mutex
	estado    int // durmiendo/observando/cortando
	cliente *Cliente
}

type Cliente struct {
	name string
}

func (c *Cliente) String() string {
	return fmt.Sprintf("%p", c)[7:]
}

func NuevoBarbero() (b *Barbero) {
	return &Barbero{
		name:  "Jos",
		estado: durmiendo,
	}
}

// El barbero duerme hasta que un cliente lo haga despertarSleeps - wait for wakers to wake him up
func barbero(b *Barbero, can_esperar chan *Cliente, can_despertar chan *Cliente) {
	for {
		b.Lock() //permanecerá bloqueado hsta que un cliente aparezca
		defer b.Unlock() //defer para asegurarse de que el barbero se bloquee despues de ejecutarse alguna opcion del select
		b.estado = observando
		b.cliente = nil //apareció un cliente. Nill porque cliente es un objeto de la esturctura barbero, la cual es mutex, no se puede inicializar

		// Barbero obserba si hay clientes esperando
		fmt.Printf("Observando si hay clientes en espera: %d\n", len(can_esperar))
		time.Sleep(time.Millisecond * 100)
		select {
		case c := <- can_esperar: //cuando llega un cliente del canal esperar turmo
			CortarPelo(c, b)
			b.Unlock() //despues de cortarle el pelo el barbero se desbloquea
		default: // cuando no hay nadie esperando por defecto llegara un primer cliente que despertará al barbero
			fmt.Printf("Barbero durmiendo - %s\n", b.cliente)
			b.estado = durmiendo
			b.cliente = nil //aparece el primer cliente
			b.Unlock() //el barbero se desbloquea
			c := <-can_despertar //cuando llega un cliente del canal despertar al barbero
			b.Lock() //el barbero de bloquea
			fmt.Printf("Despertado por el cliente %s\n", c)
			CortarPelo(c, b)
			b.Unlock() //de desbloque cuando termina
		}
	}
}

func CortarPelo(c *Cliente, b *Barbero) { //cambiará el estado del barbero a cortando para que sea visible para los clientes
	b.estado = cortando
	b.cliente = c //el cliente que llega 
	b.Unlock()
	fmt.Printf("Atendiendo  %s al cliente \n", c)
	time.Sleep(time.Millisecond * 100)
	b.Lock()
	wg.Done() // el contador del waitgroup decrementa
	b.cliente = nil
}

// el cliente sale de la peluqueria si las sillas estan llenas
// si no, pasa al canal de espera
func cliente(c *Cliente, b *Barbero, can_esperar chan<- *Cliente, can_despertar chan<- *Cliente) {
	// llega
	time.Sleep(time.Millisecond * 50)
	// mira el estado del barbero
	b.Lock()
	fmt.Printf("Cliente %s ve  %s al barbero | Esperando: %d, w %d - Atendiendose: %s\n",
		c, stateLog[b.estado], len(can_esperar), len(can_despertar), b.cliente)
	switch b.estado { //cuando un cliente entra observa al barbero
	case durmiendo: //si barbero duerme
		select {
		case can_despertar <- c: //el cliente pasa al canal despertar a levantarlo
		default: //por defecto lo primero que hace un cliente al entrar es sentarse (si hay sillas libres)
			select {
			case can_esperar <- c: // el cliente pasa al canal de espera
			default:
				wg.Done() //se decrementa el contador de waitgoup
			}
		}
	case cortando: //si el barbero corta
		select {
		case can_esperar <- c: //el cliente pasa al canal de espera
		default:   // Full waiting room, leave shop
			wg.Done() //decrementa
		}
	case observando: //el programa entra en panico
		panic("El cliente mira si el barbero esta ocupado, cuando el barbero está revisando si hay clientes esperando ")
	}
	b.Unlock()
}

func main() {
	b := NuevoBarbero() 
	b.name = "Rocky"
	Sala_de_espera:= make(chan *Cliente, 5) // 5 sillas
	Despierta_barbero := make(chan *Cliente, 1)      // solo 1 cliente puede sedpertar al barbero y ser antendo en la silla del barbero
	go barbero(b, Sala_de_espera, Despierta_barbero)    //parametros(el barbero, el canal de espera, el canal despertar barbero)

	time.Sleep(time.Millisecond * 100)
	wg = new(sync.WaitGroup)  //definiendo el waitgroup
	n := 10
	wg.Add(10)
	// generando los clientes n = numero de clientes
	for i := 0; i < n; i++ {
		time.Sleep(time.Millisecond * 50)
		c := new(Cliente)
		go cliente(c, b, Sala_de_espera, Despierta_barbero)
	}
	wg.Wait()
	fmt.Println("No hay mas clientes")
}