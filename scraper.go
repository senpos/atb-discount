package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type DiscountPageResponse struct {
	Markup   string `json:"markup"`
	NextPage bool   `json:"next_page"`
}

type DiscountItem struct {
	Title        string  `json:"title"`
	URL          string  `json:"url"`
	ImageURL     string  `json:"imageUrl"`
	CurrentPrice float64 `json:"currentPrice"`
	OldPrice     float64 `json:"oldPrice"`
	Discount     int     `json:"discount"`
	Unit         string  `json:"unit"`
}

func (s *Server) GetDiscountItems(ctx context.Context) ([]DiscountItem, error) {
	val, err := s.cache.Get(ctx, "items", func(ctx context.Context) (interface{}, error) {
		return s.scrapeDiscountItems()
	})
	if err != nil {
		return nil, err
	}
	return val.([]DiscountItem), nil
}

//goland:noinspection ALL
func (s *Server) scrapeDiscountItems() ([]DiscountItem, error) {
	var items []DiscountItem
	page := 0
	url := fmt.Sprintf("%s/%s", s.ATBBaseURL, "shop/catalog/wloadmore/")
	for {
		request, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		request.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:106.0) Gecko/20100101 Firefox/106.0")
		query := request.URL.Query()
		query.Add("customCat", "economy")
		query.Add("store", "1154")
		query.Add("page", strconv.Itoa(page))
		request.URL.RawQuery = query.Encode()

		response, err := s.HttpClient.Do(request)
		if err != nil {
			return nil, fmt.Errorf("could not fetch page %d: %v", page, err)
		}
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}

		var discountPageResponse DiscountPageResponse
		err = json.Unmarshal(body, &discountPageResponse)
		if err != nil {
			return nil, err
		}

		pageItems, err := s.parsePage(discountPageResponse.Markup)
		if err != nil {
			return nil, err
		}
		items = append(items, pageItems...)

		page = page + 1

		if !discountPageResponse.NextPage {
			break
		}
	}
	if len(items) < 1 {
		log.Printf("scraped 0 items from %s, does not seem right\n", url)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Discount > items[j].Discount })
	return items, nil
}

func (s *Server) parsePage(html string) ([]DiscountItem, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	var items []DiscountItem
	doc.Find(".catalog-item").Each(func(_ int, selector *goquery.Selection) {
		priceNode := selector.Find(".catalog-item__product-price").First()
		if !priceNode.HasClass("product-price--sale") {
			return
		}

		oldPrice, _ := strconv.ParseFloat(priceNode.Find(".product-price__bottom").First().AttrOr("value", "0"), 64)
		currentPrice, _ := strconv.ParseFloat(priceNode.Find(".product-price__top").First().AttrOr("value", "0"), 64)
		discount := int(math.Round((oldPrice - currentPrice) / oldPrice * 100))
		unit := strings.TrimSpace(priceNode.Find("abbr.product-price__currency-abbr").First().Text())

		titleNode := selector.Find(".catalog-item__title > a").First()
		title := strings.TrimSpace(titleNode.Text())
		url := titleNode.AttrOr("href", "")
		if url != "" {
			url = s.ATBBaseURL + url
		}
		imageURL := selector.Find(".catalog-item__img").First().AttrOr("src", "")
		
		item := DiscountItem{
			Title:        title,
			URL:          url,
			ImageURL:     imageURL,
			CurrentPrice: currentPrice,
			OldPrice:     oldPrice,
			Discount:     discount,
			Unit:         unit,
		}
		items = append(items, item)
	})
	return items, nil
}
