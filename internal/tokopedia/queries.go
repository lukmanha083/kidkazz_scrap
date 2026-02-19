package tokopedia

import (
	"fmt"
	"net/url"
)

const graphQLEndpoint = "https://gql.tokopedia.com/graphql/SearchProductQueryV4"

const searchProductQuery = `query SearchProductQueryV4($params: String!) {
  ace_search_product_v4(params: $params) {
    header {
      totalData
      totalDataText
      responseCode
      keywordProcess
    }
    data {
      products {
        id
        name
        price {
          text
          number
        }
        originalPrice
        discountPercentage
        imageUrl {
          300
        }
        url
        shop {
          id
          name
          city
          isOfficial
        }
        ratingAverage
        countReview
      }
    }
  }
}`

const pdpGetLayoutQuery = `query PDPGetLayoutQuery($shopDomain: String, $productKey: String, $layoutID: String, $apiVersion: Float, $extParam: String) {
  pdpGetLayout(shopDomain: $shopDomain, productKey: $productKey, layoutID: $layoutID, apiVersion: $apiVersion, extParam: $extParam) {
    name
    pdpSession
    basicInfo {
      id
      shopID
      shopName
      minOrder
      maxOrder
      weight
      weightUnit
      condition
      status
      url
      needPrescription
      catalogID
      isLeasing
      isBlacklisted
      isTokoNow
      menu {
        id
        name
        url
      }
      category {
        id
        name
        title
        isAdult
        breadcrumbURL
        detail {
          id
          name
          breadcrumbURL
          isAdult
        }
      }
    }
    components {
      name
      type
      data {
        ... on ProductMedia {
          media {
            type
            urlOriginal
            urlThumbnail
            url300
          }
        }
        ... on ProductContent {
          name
          price {
            value
            currency
          }
          campaign {
            campaignID
            campaignType
            campaignTypeName
            percentageAmount
            originalPrice
            discountedPrice
            stock {
              useStock
              value
              stockWording
            }
          }
        }
        ... on ProductReview {
          rating
          totalReview
        }
      }
    }
  }
}`

// BuildSearchParams constructs the URL-encoded params string for SearchProductQueryV4.
func BuildSearchParams(keyword string, page, rows int) string {
	start := (page - 1) * rows
	params := url.Values{}
	params.Set("q", keyword)
	params.Set("start", fmt.Sprintf("%d", start))
	params.Set("rows", fmt.Sprintf("%d", rows))
	params.Set("page", fmt.Sprintf("%d", page))
	params.Set("ob", "23") // best match
	params.Set("device", "desktop")
	params.Set("source", "search")
	return params.Encode()
}
