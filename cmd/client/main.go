package main

func main() {
	vpn := NewVPNClient()
	defer vpn.DB.Close()

	vpn.Run()
}
