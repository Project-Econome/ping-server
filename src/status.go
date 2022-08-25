package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mcstatus-io/mcutil"
)

type StatusResponse struct {
	Online      bool   `json:"online"`
	Host        string `json:"host"`
	Port        uint16 `json:"port"`
	EULABlocked bool   `json:"eula_blocked"`
}

type JavaStatusResponse struct {
	StatusResponse
	Version *JavaVersion `json:"version"`
	Players JavaPlayers  `json:"players"`
	MOTD    MOTD         `json:"motd"`
	Icon    *string      `json:"icon"`
	Mods    []Mod        `json:"mods"`
}

type BedrockStatusResponse struct {
	StatusResponse
	Version  *BedrockVersion `json:"version"`
	Players  *BedrockPlayers `json:"players"`
	MOTD     *MOTD           `json:"motd"`
	Gamemode *string         `json:"gamemode"`
	ServerID *string         `json:"server_id"`
	Edition  *string         `json:"edition"`
}

type JavaVersion struct {
	NameRaw   string `json:"name_raw"`
	NameClean string `json:"name_clean"`
	NameHTML  string `json:"name_html"`
	Protocol  int    `json:"protocol"`
}

type BedrockVersion struct {
	Name     *string `json:"name"`
	Protocol *int64  `json:"protocol"`
}

type JavaPlayers struct {
	Online int      `json:"online"`
	Max    int      `json:"max"`
	List   []Player `json:"list"`
}

type BedrockPlayers struct {
	Online *int64 `json:"online"`
	Max    *int64 `json:"max"`
}

type Player struct {
	UUID      string `json:"uuid"`
	NameRaw   string `json:"name_raw"`
	NameClean string `json:"name_clean"`
	NameHTML  string `json:"name_html"`
}

type MOTD struct {
	Raw   string `json:"raw"`
	Clean string `json:"clean"`
	HTML  string `json:"html"`
}

type Mod struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func GetJavaStatus(host string, port uint16) (string, *time.Duration, error) {
	cacheKey := fmt.Sprintf("java:%s-%d", host, port)

	exists, value, ttl, err := r.GetCacheString(cacheKey)

	if exists {
		return value, &ttl, err
	}

	response, err := json.Marshal(FetchJavaStatus(host, port))

	if err != nil {
		return "", nil, err
	}

	if err := r.Set(cacheKey, response, config.Cache.JavaCacheDuration); err != nil {
		return "", nil, err
	}

	return string(response), nil, nil
}

func GetBedrockStatus(host string, port uint16) (string, *time.Duration, error) {
	cacheKey := fmt.Sprintf("bedrock:%s-%d", host, port)

	exists, value, ttl, err := r.GetCacheString(cacheKey)

	if exists {
		return value, &ttl, err
	}

	response, err := json.Marshal(FetchBedrockStatus(host, port))

	if err != nil {
		return "", nil, err
	}

	if err := r.Set(cacheKey, response, config.Cache.BedrockCacheDuration); err != nil {
		return "", nil, err
	}

	return string(response), nil, nil
}

func GetServerIcon(host string, port uint16) ([]byte, *time.Duration, error) {
	cacheKey := fmt.Sprintf("icon:%s-%d", host, port)

	exists, value, ttl, err := r.GetCacheBytes(cacheKey)

	if exists {
		return value, &ttl, err
	}

	icon := defaultIconBytes

	status, err := mcutil.Status(host, port)

	if err == nil && status.Favicon != nil && strings.HasPrefix(*status.Favicon, "data:image/png;base64,") {
		data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(*status.Favicon, "data:image/png;base64,"))

		if err != nil {
			return nil, nil, err
		}

		icon = data
	}

	if err := r.Set(cacheKey, icon, config.Cache.IconCacheDuration); err != nil {
		return nil, nil, err
	}

	return icon, nil, nil
}

func FetchJavaStatus(host string, port uint16) interface{} {
	status, err := mcutil.Status(host, port)

	if err != nil {
		statusLegacy, err := mcutil.StatusLegacy(host, port)

		if err != nil {
			return StatusResponse{
				Online:      false,
				Host:        host,
				Port:        port,
				EULABlocked: IsBlockedAddress(host),
			}
		}

		response := JavaStatusResponse{
			StatusResponse: StatusResponse{
				Online:      true,
				Host:        host,
				Port:        port,
				EULABlocked: IsBlockedAddress(host),
			},
			Version: nil,
			Players: JavaPlayers{
				Online: statusLegacy.Players.Online,
				Max:    statusLegacy.Players.Max,
				List:   make([]Player, 0),
			},
			MOTD: MOTD{
				Raw:   statusLegacy.MOTD.Raw,
				Clean: statusLegacy.MOTD.Clean,
				HTML:  statusLegacy.MOTD.HTML,
			},
			Icon: nil,
			Mods: make([]Mod, 0),
		}

		if statusLegacy.Version != nil {
			response.Version = &JavaVersion{
				NameRaw:   statusLegacy.Version.NameRaw,
				NameClean: statusLegacy.Version.NameClean,
				NameHTML:  statusLegacy.Version.NameHTML,
				Protocol:  statusLegacy.Version.Protocol,
			}
		}

		return response
	}

	playerList := make([]Player, 0)

	if status.Players.Sample != nil {
		for _, player := range status.Players.Sample {
			playerList = append(playerList, Player{
				UUID:      player.ID,
				NameRaw:   player.NameRaw,
				NameClean: player.NameClean,
				NameHTML:  player.NameHTML,
			})
		}
	}

	modList := make([]Mod, 0)

	if status.ModInfo != nil {
		for _, mod := range status.ModInfo.Mods {
			modList = append(modList, Mod{
				Name:    mod.ID,
				Version: mod.Version,
			})
		}
	}

	return JavaStatusResponse{
		StatusResponse: StatusResponse{
			Online:      true,
			Host:        host,
			Port:        port,
			EULABlocked: IsBlockedAddress(host),
		},
		Version: &JavaVersion{
			NameRaw:   status.Version.NameRaw,
			NameClean: status.Version.NameClean,
			NameHTML:  status.Version.NameHTML,
			Protocol:  status.Version.Protocol,
		},
		Players: JavaPlayers{
			Online: status.Players.Online,
			Max:    status.Players.Max,
			List:   playerList,
		},
		MOTD: MOTD{
			Raw:   status.MOTD.Raw,
			Clean: status.MOTD.Clean,
			HTML:  status.MOTD.HTML,
		},
		Icon: status.Favicon,
		Mods: modList,
	}
}

func FetchBedrockStatus(host string, port uint16) interface{} {
	status, err := mcutil.StatusBedrock(host, port)

	if err != nil {
		return StatusResponse{
			Online:      false,
			Host:        host,
			Port:        port,
			EULABlocked: IsBlockedAddress(host),
		}
	}

	response := BedrockStatusResponse{
		StatusResponse: StatusResponse{
			Online:      true,
			Host:        host,
			Port:        port,
			EULABlocked: IsBlockedAddress(host),
		},
		Version:  nil,
		Players:  nil,
		MOTD:     nil,
		Gamemode: status.Gamemode,
		ServerID: status.ServerID,
		Edition:  status.Edition,
	}

	if status.Version != nil {
		if response.Version == nil {
			response.Version = &BedrockVersion{
				Name:     nil,
				Protocol: nil,
			}
		}

		response.Version.Name = status.Version
	}

	if status.ProtocolVersion != nil {
		if response.Version == nil {
			response.Version = &BedrockVersion{
				Name:     nil,
				Protocol: nil,
			}
		}

		response.Version.Protocol = status.ProtocolVersion
	}

	if status.OnlinePlayers != nil {
		if response.Players == nil {
			response.Players = &BedrockPlayers{
				Online: nil,
				Max:    nil,
			}
		}

		response.Players.Online = status.OnlinePlayers
	}

	if status.MaxPlayers != nil {
		if response.Players == nil {
			response.Players = &BedrockPlayers{
				Online: nil,
				Max:    nil,
			}
		}

		response.Players.Max = status.MaxPlayers
	}

	if status.MOTD != nil {
		response.MOTD = &MOTD{
			Raw:   status.MOTD.Raw,
			Clean: status.MOTD.Clean,
			HTML:  status.MOTD.HTML,
		}
	}

	return response
}
