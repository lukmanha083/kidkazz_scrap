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
		processTime
		responseCode
		errorMessage
		additionalParams
		keywordProcess
		__typename
		}
		data {
		isQuerySafe
		ticker {
			text
			query
			typeId
			__typename
		}
		redirection {
			redirectUrl
			departmentId
			__typename
		}
		related {
			relatedKeyword
			otherRelated {
			keyword
			url
			product {
				id
				name
				price
				imageUrl
				rating
				countReview
				url
				priceStr
				wishlist
				shop {
				city
				isOfficial
				isPowerBadge
				__typename
				}
				ads {
				id
				productClickUrl
				productWishlistUrl
				shopClickUrl
				productViewUrl
				__typename
				}
				__typename
			}
			__typename
			}
			__typename
		}
		suggestion {
			currentKeyword
			suggestion
			suggestionCount
			instead
			insteadCount
			query
			text
			__typename
		}
		products {
			id
			name
			ads {
			id
			productClickUrl
			productWishlistUrl
			productViewUrl
			__typename
			}
			badges {
			title
			imageUrl
			show
			__typename
			}
			category: departmentId
			categoryBreadcrumb
			categoryId
			categoryName
			countReview
			discountPercentage
			gaKey
			imageUrl
			labelGroups {
			position
			title
			type
			__typename
			}
			originalPrice
			price
			priceRange
			rating
			shop {
			id
			name
			url
			city
			isOfficial
			isPowerBadge
			__typename
			}
			url
			wishlist
			sourceEngine: source_engine
			__typename
		}
		__typename
		}
		__typename
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

// Sort order constants for Tokopedia search.
const (
	SortBestMatch  = 23
	SortBestSeller = 5 // most reviews (ulasan/terlaris)
	SortNewest     = 9
	SortPriceAsc   = 3
	SortPriceDesc  = 4
)

// BuildSearchParams constructs the URL-encoded params string for SearchProductQueryV4.
func BuildSearchParams(keyword string, page, rows, orderBy int) string {
	start := (page - 1) * rows
	params := url.Values{}
	params.Set("q", keyword)
	params.Set("start", fmt.Sprintf("%d", start))
	params.Set("rows", fmt.Sprintf("%d", rows))
	params.Set("page", fmt.Sprintf("%d", page))
	params.Set("ob", fmt.Sprintf("%d", orderBy))
	params.Set("device", "desktop")
	params.Set("source", "search")
	return params.Encode()
}
