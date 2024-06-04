package backs

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
)

func Who(s *discordgo.Session, info BackInfo, backFiles *BackMapping) error {

	file, err := chooseBack(backFiles)
	if err != nil {
		fmt.Println("Could not choose a back!!! - CRITICAL: ", err)
		return err
	}

	fmt.Println("BACK CHOSEN: ", file)

	back, err := loadBack(file)
	if err != nil {
		fmt.Println("Could not acknowledge back!!! - CRITICAL: ", err)
		return err
	}
	err = playBack(s, info, back)
	return err
}

func playBack(s *discordgo.Session, info BackInfo, backBytes [][]byte) error {
	// Join the provided voice channel.
	vc, err := s.ChannelVoiceJoin(info.VoiceState.GuildID, info.VoiceState.ChannelID, false, false)

	defer vc.Disconnect()

	if err != nil {
		fmt.Println("error joining channel: ", err)
		return err
	}

	// Sleep for a specified amount of time before playing the sound
	time.Sleep(50 * time.Millisecond)

	err = vc.Speaking(true)
	if err != nil {
		fmt.Println("I have no mouth but I must back: ", err)
		return err
	}

	// Who?
	for _, buff := range backBytes {
		vc.OpusSend <- buff
	}

	// Stop speaking
	vc.Speaking(false)

	// Sleep for a specificed amount of time before ending.
	time.Sleep(50 * time.Millisecond)

	return nil
}

func loadBack(backPath string) ([][]byte, error) {
	buffer := make([][]byte, 0)
	file, err := os.Open(backPath)
	if err != nil {
		fmt.Println("Error opening dca file :", err)
		return [][]byte{}, err
	}

	var opuslen int16

	for {
		// Read opus frame length from dca file.
		err = binary.Read(file, binary.LittleEndian, &opuslen)

		// If this is the end of the file, just return.
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			err := file.Close()
			if err != nil {
				return [][]byte{}, err
			}
			return buffer, nil
		}

		if err != nil {
			fmt.Println("Error reading from dca file :", err)
			return [][]byte{}, err
		}

		// Read encoded pcm from dca file.
		InBuf := make([]byte, opuslen)
		err = binary.Read(file, binary.LittleEndian, &InBuf)

		// Should not be any end of file errors
		if err != nil {
			fmt.Println("Error reading from dca file :", err)
			return [][]byte{}, err
		}

		// Append encoded pcm data to the buffer.
		buffer = append(buffer, InBuf)
	}
}
