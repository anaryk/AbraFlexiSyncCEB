package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
)

type Config struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Stats struct {
	Created int `xml:"created"`
	Updated int `xml:"updated"`
	Deleted int `xml:"deleted"`
	Skipped int `xml:"skipped"`
	Failed  int `xml:"failed"`
}

type Response struct {
	Success bool  `xml:"success"`
	Stats   Stats `xml:"stats"`
}

type CentralConfig struct {
	Directories             []string `yaml:"directories"`
	PaymentOrderDirectories []string `yaml:"payment_order_directories"`
	LogToFile               bool     `yaml:"log_to_file"`
}

type PaymentOrder struct {
	ID         string `json:"id"`
	LastUpdate string `json:"lastUpdate"`
	DatSplat   string `json:"datSplat"`
	Mena       string `json:"mena"`
}

type PaymentOrderResponse struct {
	Winstrom struct {
		Version       string         `json:"@version"`
		PaymentOrders []PaymentOrder `json:"prikaz-k-uhrade"`
	} `json:"winstrom"`
}

func loadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func loadCentralConfig(configPath string) (*CentralConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var centralConfig CentralConfig
	err = yaml.Unmarshal(data, &centralConfig)
	if err != nil {
		return nil, err
	}

	return &centralConfig, nil
}

func processFiles(directory string, config *Config) error {
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && (strings.HasSuffix(info.Name(), ".GPC") || strings.HasSuffix(info.Name(), ".gpc")) {
			fileContent, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			req, err := http.NewRequest("POST", config.URL, bytes.NewReader(fileContent))
			if err != nil {
				return err
			}

			auth := config.Username + ":" + config.Password
			encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
			req.Header.Add("Authorization", "Basic "+encodedAuth)
			req.Header.Add("Content-Type", "application/octet-stream")

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				var response Response
				err := xml.NewDecoder(resp.Body).Decode(&response)
				if err != nil {
					return err
				}

				if response.Success {
					newPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".nahrano"
					err := os.Rename(path, newPath)
					if err != nil {
						return err
					}
					log.Info().Msgf("Successfully processed (Uploaded to AbraCloud) and renamed file: %s", path)
					log.Info().Msgf("Stats - Created: %d, Updated: %d, Deleted: %d, Skipped: %d, Failed: %d",
						response.Stats.Created, response.Stats.Updated, response.Stats.Deleted, response.Stats.Skipped, response.Stats.Failed)
				} else {
					log.Warn().Msgf("Failed to process file: %s, success: %t", path, response.Success)
				}
			} else {
				log.Warn().Msgf("Failed to process file: %s, status code: %d", path, resp.StatusCode)
			}
		}

		return nil
	})

	return err
}

func processPaymentOrders(directory string, config *Config) error {
	req, err := http.NewRequest("GET", config.URL+"/prikaz-k-uhrade/(stavPrikazK='elPrikazStav.vytvoren').json", nil)
	if err != nil {
		return err
	}

	auth := config.Username + ":" + config.Password
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
	req.Header.Add("Authorization", "Basic "+encodedAuth)
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Warn().Msgf("Failed to check for payment orders on %s, with status code: %d", config.URL+"/prikaz-k-uhrade/(stavPrikazK='elPrikazStav.neodeslan').json", resp.StatusCode)
		return nil
	}

	var paymentOrderResponse PaymentOrderResponse
	err = json.NewDecoder(resp.Body).Decode(&paymentOrderResponse)
	if err != nil {
		return err
	}

	if len(paymentOrderResponse.Winstrom.PaymentOrders) == 0 {
		log.Info().Msgf("No payment orders to download in directory %s", directory)
		return nil
	}

	for _, order := range paymentOrderResponse.Winstrom.PaymentOrders {
		orderID := order.ID
		orderURL := fmt.Sprintf("%s/prikaz-k-uhrade/%s/stazeni?dat-splat-z-hlavicky=true", config.URL, orderID)
		req, err := http.NewRequest("GET", orderURL, nil)
		if err != nil {
			return err
		}

		req.Header.Add("Authorization", "Basic "+encodedAuth)
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			fileName := fmt.Sprintf("%s-%s.kpc", time.Now().Format("20060102150405"), orderID)
			var filePath string
			if runtime.GOOS == "windows" {
				filePath = filepath.Join(directory+"\\kpc", fileName)
				err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
				if err != nil {
					return err
				}
			} else {
				filePath = filepath.Join(directory+"/kpc", fileName)
				err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
				if err != nil {
					return err
				}
			}
			outFile, err := os.Create(filePath)
			if err != nil {
				return err
			}
			defer outFile.Close()

			_, err = io.Copy(outFile, resp.Body)
			if err != nil {
				return err
			}
			log.Info().Msgf("Successfully downloaded payment order %s to %s", orderID, filePath)
		} else {
			log.Warn().Msgf("Failed to download payment order %s, status code: %d", orderID, resp.StatusCode)
		}
	}

	return nil
}

func processDirectories(directories []string, paymentOrderDirs []string) {
	for _, dir := range directories {
		configPath := filepath.Join(dir, "config.yaml")
		config, err := loadConfig(configPath)
		if err != nil {
			log.Warn().Msgf("Skipping directory %s: %v", dir, err)
			continue
		}

		log.Info().Msgf("Starting processing for directory %s.", dir)
		err = processFiles(dir, config)
		if err != nil {
			log.Error().Msgf("Error processing directory %s: %v", dir, err)
		} else {
			log.Info().Msgf("Processing completed successfully for directory %s.", dir)
		}
	}

	for _, dir := range paymentOrderDirs {
		configPath := filepath.Join(dir, "config.yaml")
		config, err := loadConfig(configPath)
		if err != nil {
			log.Warn().Msgf("Skipping payment order directory %s: %v", dir, err)
			continue
		}

		log.Info().Msgf("Starting processing of payment orders for directory %s.", dir)
		err = processPaymentOrders(dir, config)
		if err != nil {
			log.Error().Msgf("Error processing payment order directory %s: %v", dir, err)
		} else {
			log.Info().Msgf("Processing of payment orders completed successfully for directory %s.", dir)
		}
	}
}

func getDirectories() ([]string, []string) {
	centralConfigPath := filepath.Join(filepath.Dir(os.Args[0]), "central_config.yaml")
	centralConfig, err := loadCentralConfig(centralConfigPath)
	if err == nil {
		log.Info().Msg("Using central config for directories.")
		return centralConfig.Directories, centralConfig.PaymentOrderDirectories
	}
	log.Error().Msgf("Loading config error: %v", err)

	log.Info().Msg("Using environment variables for directories.")
	var directories, paymentOrderDirs []string
	for i := 1; ; i++ {
		dir := os.Getenv(fmt.Sprintf("DIR_%d", i))
		if dir == "" {
			break
		}
		directories = append(directories, dir)
	}

	for i := 1; ; i++ {
		dir := os.Getenv(fmt.Sprintf("PAYMENT_ORDER_DIR_%d", i))
		if dir == "" {
			break
		}
		paymentOrderDirs = append(paymentOrderDirs, dir)
	}

	return directories, paymentOrderDirs
}

func main() {
	centralConfigPath := filepath.Join(filepath.Dir(os.Args[0]), "central_config.yaml")
	centralConfig, err := loadCentralConfig(centralConfigPath)

	var logFile *os.File
	if err == nil && centralConfig.LogToFile {
		logFile, err = os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			fmt.Printf("Failed to open log file: %v", err)
			return
		}
		defer logFile.Close()
		multi := zerolog.MultiLevelWriter(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}, logFile)
		log.Logger = zerolog.New(multi).With().Timestamp().Logger()
	} else {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).With().Timestamp().Logger()
	}

	zerolog.TimeFieldFormat = time.RFC3339

	log.Info().Msg("Starting initial processing of directories.")
	directories, paymentOrderDirs := getDirectories()
	processDirectories(directories, paymentOrderDirs)

	log.Info().Msg("Starting ticker for scheduled processing of directories.")
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		log.Info().Msg("Starting scheduled processing of directories.")
		processDirectories(directories, paymentOrderDirs)
	}
}
