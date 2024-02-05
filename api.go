package main

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"time"
	// "bufio"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/hashicorp/golang-lru/v2/expirable"
)

var ShortUrlDigit int = 9

// make cache with 10ms TTL and 5 max keys
var cache *expirable.LRU[string, string] = expirable.NewLRU[string, string](256, nil, time.Hour*1)

type URLDocument struct {
	URL string `json:"url" dynamodbav:"url"`
	ID  string `json:"url_id" dynamodbav:"url_id"`
}

func (urldocument URLDocument) GetKey() (map[string]types.AttributeValue, error) {
	m := map[string]types.AttributeValue{}
	id, err := attributevalue.Marshal(urldocument.ID)
	if err != nil {
		zapLogger.Error(fmt.Sprintf("Couldn't parse id. Here's why: %v", err))
		return m, err
	}

	return map[string]types.AttributeValue{"url_id": id}, nil
}

func getID(node *Node) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		if node == nil {
			c.JSON(http.StatusNotFound, "")
			return
		}
		c.JSON(http.StatusOK, node.node)
	})
}

func setURL(dbclient *dynamodb.Client, node *Node) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		if dbclient == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"err": "database failed to connect"})
			return
		}
		table := Getenv("TABLE_NAME", "urlshorter")
		hostname := Getenv("URL_DOMAIN", "127.0.0.1:8080")
		var err error
		urldocument := URLDocument{}
		err = c.ShouldBindBodyWith(&urldocument, binding.JSON)
		if _, err := url.ParseRequestURI(urldocument.URL); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"err": fmt.Sprintf("url %s is invalid", urldocument.URL)})
			return
		}
		id, err := node.Generate()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"err": "failed to generate id"})
			return
		}
		urldocument.ID = id.Base62(ShortUrlDigit)
		err = AddURLToTable(dbclient, table, urldocument)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"err": fmt.Sprintf("failed to add to table, err: %v", err)})
			return
		}
		c.JSON(http.StatusOK, gin.H{"url": urldocument.URL, "shortenUrl": fmt.Sprintf("http://%s/%s", hostname, urldocument.ID)})
	})
}

func proxy(dbclient *dynamodb.Client) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		r, err := regexp.Compile(`[a-zA-Z0-9]{9}`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"err": fmt.Sprintf("url parse error %v", err)})
			return
		}
		id := c.Param("regex")
		if !r.MatchString(id) || len(id) > ShortUrlDigit {
			//Write error header and body no match
			c.JSON(http.StatusBadRequest, gin.H{"err": "invalid shorten url"})
			return
		}

		proxyUrl, ok := cache.Get(id)
		if !ok {
			if dbclient == nil {
				c.JSON(http.StatusInternalServerError, gin.H{"err": "database failed to connect"})
				return
			}
			table := Getenv("TABLE_NAME", "urlshorter")
			urldocument, err := GetURLFromTable(c.Request.Context(), dbclient, table, id)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"err": fmt.Sprintf("failed to get shorten url from table, err: %v", err)})
				return
			}
			proxyUrl = urldocument.URL
			cache.Add(urldocument.ID, urldocument.URL)
		}

		
		_, err = url.ParseRequestURI(proxyUrl)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"err": fmt.Sprintf("url %s is invalid", proxyUrl)})
			return
		}
		c.Redirect(http.StatusMovedPermanently, proxyUrl)

		// // step 1: resolve proxy address, change scheme and host in requets
		// req := c.Request
		// proxy, err := url.ParseRequestURI(url.URL)
		// if err != nil {
		// 	c.JSON(http.StatusBadRequest, gin.H{"err": fmt.Sprintf("url %s is invalid", url.URL)})
		// }
		// req.URL.Scheme = proxy.Scheme
		// req.URL.Host = proxy.Host

		// // step 2: use http.Transport to do request to real server.
		// transport := http.DefaultTransport
		// resp, err := transport.RoundTrip(req)
		// if err != nil {
		// 	c.JSON(http.StatusInternalServerError, gin.H{"err": fmt.Sprintf("error in roundtrip: %v", err)})
		// 	return
		// }

		// // step 3: return real server response to upstream.
		// for k, vv := range resp.Header {
		// 	for _, v := range vv {
		// 		c.Header(k, v)
		// 	}
		// }
		// defer resp.Body.Close()
		// bufio.NewReader(resp.Body).WriteTo(c.Writer)
		// return
	})
}
