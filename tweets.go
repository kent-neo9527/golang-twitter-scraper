package twitterscraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// GetTweets returns channel with tweets for a given user.
func (s *Scraper) GetTweets(ctx context.Context, user string, maxTweetsNbr int) <-chan *TweetResult {
	return getTweetTimeline(ctx, user, maxTweetsNbr, s.FetchTweets)
}

// FetchTweets gets tweets for a given user, via the Twitter frontend API.
func (s *Scraper) FetchTweets(user string, maxTweetsNbr int, cursor string) ([]*Tweet, string, error) {
	userID, err := s.GetUserIDByScreenName(user)
	if err != nil {
		return nil, "", err
	}

	if s.isOpenAccount {
		return s.FetchTweetsByUserIDLegacy(userID, maxTweetsNbr, cursor)
	}
	return s.FetchTweetsByUserID(userID, maxTweetsNbr, cursor)
}

// FetchTweetsByUserID gets tweets for a given userID, via the Twitter frontend GraphQL API.
func (s *Scraper) FetchTweetsByUserID(userID string, maxTweetsNbr int, cursor string) ([]*Tweet, string, error) {
	if maxTweetsNbr > 200 {
		maxTweetsNbr = 200
	}

	req, err := s.newRequest("GET", "https://twitter.com/i/api/graphql/UGi7tjRPr-d_U3bCPIko5Q/UserTweets")
	if err != nil {
		return nil, "", err
	}

	variables := map[string]interface{}{
		"userId":                                 userID,
		"count":                                  maxTweetsNbr,
		"includePromotedContent":                 false,
		"withQuickPromoteEligibilityTweetFields": false,
		"withVoice":                              true,
		"withV2Timeline":                         true,
	}
	features := map[string]interface{}{
		"rweb_lists_timeline_redesign_enabled":                              true,
		"responsive_web_graphql_exclude_directive_enabled":                  true,
		"verified_phone_label_enabled":                                      false,
		"creator_subscriptions_tweet_preview_api_enabled":                   true,
		"responsive_web_graphql_timeline_navigation_enabled":                true,
		"responsive_web_graphql_skip_user_profile_image_extensions_enabled": false,
		"tweetypie_unmention_optimization_enabled":                          true,
		"vibe_api_enabled":                                                        true,
		"responsive_web_edit_tweet_api_enabled":                                   true,
		"graphql_is_translatable_rweb_tweet_is_translatable_enabled":              true,
		"view_counts_everywhere_api_enabled":                                      true,
		"longform_notetweets_consumption_enabled":                                 true,
		"tweet_awards_web_tipping_enabled":                                        false,
		"freedom_of_speech_not_reach_fetch_enabled":                               true,
		"standardized_nudges_misinfo":                                             true,
		"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled": false,
		"interactive_text_enabled":                                                true,
		"responsive_web_text_conversations_enabled":                               false,
		"longform_notetweets_rich_text_read_enabled":                              true,
		"longform_notetweets_inline_media_enabled":                                false,
		"responsive_web_enhance_cards_enabled":                                    false,
	}

	if cursor != "" {
		variables["cursor"] = cursor
	}

	query := url.Values{}
	query.Set("variables", mapToJSONString(variables))
	query.Set("features", mapToJSONString(features))
	req.URL.RawQuery = query.Encode()

	var timeline timelineV2
	err = s.RequestAPI(req, &timeline)
	if err != nil {
		return nil, "", err
	}

	tweets, nextCursor := timeline.parseTweets()
	return tweets, nextCursor, nil
}

// FetchTweetsByUserIDLegacy gets tweets for a given userID, via the Twitter frontend legacy API.
func (s *Scraper) FetchTweetsByUserIDLegacy(userID string, maxTweetsNbr int, cursor string) ([]*Tweet, string, error) {
	if maxTweetsNbr > 200 {
		maxTweetsNbr = 200
	}

	req, err := s.newRequest("GET", "https://api.twitter.com/2/timeline/profile/"+userID+".json")
	if err != nil {
		return nil, "", err
	}

	q := req.URL.Query()
	q.Add("count", strconv.Itoa(maxTweetsNbr))
	q.Add("userId", userID)
	if cursor != "" {
		q.Add("cursor", cursor)
	}
	req.URL.RawQuery = q.Encode()

	var timeline timelineV1
	err = s.RequestAPI(req, &timeline)
	if err != nil {
		return nil, "", err
	}

	tweets, nextCursor := timeline.parseTweets()
	return tweets, nextCursor, nil
}

// GetTweet get a single tweet by ID.
func (s *Scraper) GetTweet(id string) (*Tweet, error) {
	if s.isOpenAccount {
		req, err := s.newRequest("GET", "https://api.twitter.com/2/timeline/conversation/"+id+".json")
		if err != nil {
			return nil, err
		}

		var timeline timelineV1
		err = s.RequestAPI(req, &timeline)
		if err != nil {
			return nil, err
		}

		tweets, _ := timeline.parseTweets()
		for _, tweet := range tweets {
			if tweet.ID == id {
				return tweet, nil
			}
		}
	} else {
		req, err := s.newRequest("GET", "https://twitter.com/i/api/graphql/VWFGPVAGkZMGRKGe3GFFnA/TweetDetail")
		if err != nil {
			return nil, err
		}

		variables := map[string]interface{}{
			"focalTweetId":                           id,
			"with_rux_injections":                    false,
			"includePromotedContent":                 true,
			"withCommunity":                          true,
			"withQuickPromoteEligibilityTweetFields": true,
			"withBirdwatchNotes":                     true,
			"withVoice":                              true,
			"withV2Timeline":                         true,
		}

		features := map[string]interface{}{
			"rweb_lists_timeline_redesign_enabled":                                    true,
			"responsive_web_graphql_exclude_directive_enabled":                        true,
			"verified_phone_label_enabled":                                            false,
			"creator_subscriptions_tweet_preview_api_enabled":                         true,
			"responsive_web_graphql_timeline_navigation_enabled":                      true,
			"responsive_web_graphql_skip_user_profile_image_extensions_enabled":       false,
			"tweetypie_unmention_optimization_enabled":                                true,
			"responsive_web_edit_tweet_api_enabled":                                   true,
			"graphql_is_translatable_rweb_tweet_is_translatable_enabled":              true,
			"view_counts_everywhere_api_enabled":                                      true,
			"longform_notetweets_consumption_enabled":                                 true,
			"tweet_awards_web_tipping_enabled":                                        false,
			"freedom_of_speech_not_reach_fetch_enabled":                               true,
			"standardized_nudges_misinfo":                                             true,
			"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled": false,
			"longform_notetweets_rich_text_read_enabled":                              true,
			"longform_notetweets_inline_media_enabled":                                true,
			"responsive_web_enhance_cards_enabled":                                    false,
		}

		query := url.Values{}
		query.Set("variables", mapToJSONString(variables))
		query.Set("features", mapToJSONString(features))
		req.URL.RawQuery = query.Encode()

		var conversation threadedConversation

		// Surprisingly, if bearerToken2 is not set, then animated GIFs are not
		// present in the response for tweets with a GIF + a photo like this one:
		// https://twitter.com/Twitter/status/1580661436132757506
		curBearerToken := s.bearerToken
		if curBearerToken != bearerToken2 {
			s.setBearerToken(bearerToken2)
		}

		err = s.RequestAPI(req, &conversation)

		if curBearerToken != bearerToken2 {
			s.setBearerToken(curBearerToken)
		}

		if err != nil {
			return nil, err
		}

		tweets := conversation.parse()
		for _, tweet := range tweets {
			if tweet.ID == id {
				return tweet, nil
			}
		}
	}
	return nil, fmt.Errorf("tweet with ID %s not found", id)
}

// {"variables":{"tweet_id":"1915209125442752818"},"queryId":"aoDbu3RHznuiSkQ9aNM67Q"}
// https://x.com/i/api/graphql/aoDbu3RHznuiSkQ9aNM67Q/CreateBookmark {"variables":{"tweet_id":"1914635552831177176"},"queryId":"aoDbu3RHznuiSkQ9aNM67Q"} {"data":{"tweet_bookmark_put":"Done"}} POST
func (s *Scraper) Bookmark(tweetID string) (*LikeAndBookmark, error) {
	createBookmark := "https://x.com/i/api/graphql/aoDbu3RHznuiSkQ9aNM67Q/CreateBookmark"

	// 使用 map 构建请求体
	requestBody := map[string]interface{}{
		"variables": map[string]string{
			"tweet_id": tweetID,
		},
		//"queryId": "aoDbu3RHznuiSkQ9aNM67Q",
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	req, err := s.newRequestWithData("POST", createBookmark, jsonData)
	if err != nil {
		return nil, err
	}
	var responseLikeAndBookmark LikeAndBookmark
	err = s.RequestAPI(req, &responseLikeAndBookmark)
	if err != nil {
		return nil, err
	}
	return &responseLikeAndBookmark, nil
}

// {"variables":{"tweet_id":"1915209125442752818"},"queryId":"lI07N6Otwv1PhnEgXILM7A"}
// https://x.com/i/api/graphql/lI07N6Otwv1PhnEgXILM7A/FavoriteTweet {"variables":{"tweet_id":"1914666028077818198"},"queryId":"lI07N6Otwv1PhnEgXILM7A"}   {"data":{"favorite_tweet":"Done"}} POST
func (s *Scraper) FavoriteTweet(tweetID string) (*LikeAndBookmark, error) {
	favoriteTweet := "https://x.com/i/api/graphql/lI07N6Otwv1PhnEgXILM7A/FavoriteTweet"
	requestBody := map[string]interface{}{
		"variables": map[string]string{
			"tweet_id": tweetID,
		},
		//"queryId": "lI07N6Otwv1PhnEgXILM7A",
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	req, err := s.newRequestWithData("POST", favoriteTweet, jsonData)
	if err != nil {
		return nil, err
	}
	var responseLikeAndBookmark LikeAndBookmark
	err = s.RequestAPI(req, &responseLikeAndBookmark)
	if err != nil {
		return nil, err
	}
	return &responseLikeAndBookmark, nil
}

//retweetUrl = 'https://twitter.com/i/api/graphql/ojPdsZsimiJrUGLR1sjUtA/CreateRetweet'; {"variables":{"tweet_id":"1915033288688996493","dark_request":false},"queryId":"ojPdsZsimiJrUGLR1sjUtA"}

func (s *Scraper) ReTweet(tweetID string) (*LikeAndBookmark, error) {
	createRetweet := "https://twitter.com/i/api/graphql/ojPdsZsimiJrUGLR1sjUtA/CreateRetweet"
	requestBody := map[string]interface{}{
		"variables": map[string]interface{}{
			"tweet_id":     tweetID,
			"dark_request": false,
		},
		//"queryId": "ojPdsZsimiJrUGLR1sjUtA",
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	req, err := s.newRequestWithData("POST", createRetweet, jsonData)
	if err != nil {
		return nil, err
	}
	var responseLikeAndBookmark LikeAndBookmark
	err = s.RequestAPI(req, &responseLikeAndBookmark)
	if err != nil {
		return nil, err
	}
	return &responseLikeAndBookmark, nil
}

//	https://x.com/i/api/graphql/IVdJU2Vjw2llhmJOAZy9Ow/CreateTweet
//
// {"variables":{"tweet_text":"so coooooool","reply":{"in_reply_to_tweet_id":"1915018532758209002","exclude_reply_user_ids":[]},"dark_request":false,"media":{"media_entities":[],"possibly_sensitive":false},"semantic_annotation_ids":[],"disallowed_reply_options":null},"features":{"premium_content_api_read_enabled":false,"communities_web_enable_tweet_community_results_fetch":true,"c9s_tweet_anatomy_moderator_badge_enabled":true,"responsive_web_grok_analyze_button_fetch_trends_enabled":false,"responsive_web_grok_analyze_post_followups_enabled":true,"responsive_web_jetfuel_frame":false,"responsive_web_grok_share_attachment_enabled":true,"responsive_web_edit_tweet_api_enabled":true,"graphql_is_translatable_rweb_tweet_is_translatable_enabled":true,"view_counts_everywhere_api_enabled":true,"longform_notetweets_consumption_enabled":true,"responsive_web_twitter_article_tweet_consumption_enabled":true,"tweet_awards_web_tipping_enabled":false,"responsive_web_grok_show_grok_translated_post":false,"responsive_web_grok_analysis_button_from_backend":true,"creator_subscriptions_quote_tweet_preview_enabled":false,"longform_notetweets_rich_text_read_enabled":true,"longform_notetweets_inline_media_enabled":true,"profile_label_improvements_pcf_label_in_post_enabled":true,"rweb_tipjar_consumption_enabled":true,"responsive_web_graphql_exclude_directive_enabled":true,"verified_phone_label_enabled":false,"articles_preview_enabled":true,"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false,"freedom_of_speech_not_reach_fetch_enabled":true,"standardized_nudges_misinfo":true,"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled":true,"responsive_web_grok_image_annotation_enabled":true,"responsive_web_graphql_timeline_navigation_enabled":true,"responsive_web_enhance_cards_enabled":false},"queryId":"IVdJU2Vjw2llhmJOAZy9Ow"}
// {"data":{"create_tweet":{"tweet_results":{"result":{"rest_id":"1915322686617882698","core":{"user_results":{"result":{"__typename":"User","id":"VXNlcjoxODk2NDk2NTA0Nzc0NTEyNjQx","rest_id":"1896496504774512641","affiliates_highlighted_label":{},"has_graduated_access":true,"parody_commentary_fan_label":"None","is_blue_verified":false,"profile_image_shape":"Circle","legacy":{"following":false,"can_dm":true,"can_media_tag":true,"created_at":"Mon Mar 03 09:43:13 +0000 2025","default_profile":true,"default_profile_image":false,"description":"Developer | Tech enthusiast | Focused on AI & workflow automation | Exploring how code shapes the world","entities":{"description":{"urls":[]}},"fast_followers_count":0,"favourites_count":184,"followers_count":2,"friends_count":21,"has_custom_timelines":false,"is_translator":false,"listed_count":0,"location":"","media_count":4,"name":"Kent","needs_phone_verification":false,"normal_followers_count":2,"pinned_tweet_ids_str":[],"possibly_sensitive":false,"profile_image_url_https":"https://pbs.twimg.com/profile_images/1896496578233532417/-5Z7no7G_normal.png","profile_interstitial_type":"","screen_name":"Kent236896","statuses_count":106,"translator_type":"none","verified":false,"want_retweets":false,"withheld_in_countries":[]},"tipjar_settings":{}}}},"unmention_data":{},"edit_control":{"edit_tweet_ids":["1915322686617882698"],"editable_until_msecs":"1745487088000","is_edit_eligible":false,"edits_remaining":"5"},"is_translatable":false,"views":{"state":"Enabled"},"source":"<a href=\"https://mobile.twitter.com\" rel=\"nofollow\">Twitter Web App</a>","grok_analysis_button":true,"legacy":{"bookmark_count":0,"bookmarked":false,"created_at":"Thu Apr 24 08:31:28 +0000 2025","conversation_id_str":"1915018532758209002","display_text_range":[14,26],"entities":{"hashtags":[],"symbols":[],"timestamps":[],"urls":[],"user_mentions":[{"id_str":"952599638099611650","name":"Oscar Race","screen_name":"TheOscarRace","indices":[0,13]}]},"favorite_count":0,"favorited":false,"full_text":"@TheOscarRace so coooooool","in_reply_to_screen_name":"TheOscarRace","in_reply_to_status_id_str":"1915018532758209002","in_reply_to_user_id_str":"952599638099611650","is_quote_status":false,"lang":"en","quote_count":0,"reply_count":0,"retweet_count":0,"retweeted":false,"user_id_str":"1896496504774512641","id_str":"1915322686617882698"},"unmention_info":{}}}}}}
func (s *Scraper) SendTweet(text string, retweetId string) (string, error) {
	createTweet := "https://twitter.com/i/api/graphql/a1p9RWpkYKBjWv_I3WzS-A/CreateTweet"
	// 定义 map
	variables := map[string]interface{}{
		"tweet_text":   text,
		"dark_request": false,
		"media": map[string]interface{}{
			"media_entities":     []interface{}{}, // 空数组
			"possibly_sensitive": false,
		},
		"semantic_annotation_ids": []interface{}{}, // 空数组
	}

	if retweetId != "" {
		variables["reply"] = map[string]interface{}{
			"in_reply_to_tweet_id": retweetId,
		}
	}

	features := map[string]interface{}{
		"interactive_text_enabled":                                                true,
		"longform_notetweets_inline_media_enabled":                                false,
		"responsive_web_text_conversations_enabled":                               false,
		"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled": false,
		"vibe_api_enabled":                                                        false,
		"rweb_lists_timeline_redesign_enabled":                                    true,
		"responsive_web_graphql_exclude_directive_enabled":                        true,
		"verified_phone_label_enabled":                                            false,
		"creator_subscriptions_tweet_preview_api_enabled":                         true,
		"responsive_web_graphql_timeline_navigation_enabled":                      true,
		"responsive_web_graphql_skip_user_profile_image_extensions_enabled":       false,
		"tweetypie_unmention_optimization_enabled":                                true,
		"responsive_web_edit_tweet_api_enabled":                                   true,
		"graphql_is_translatable_rweb_tweet_is_translatable_enabled":              true,
		"view_counts_everywhere_api_enabled":                                      true,
		"longform_notetweets_consumption_enabled":                                 true,
		"tweet_awards_web_tipping_enabled":                                        false,
		"freedom_of_speech_not_reach_fetch_enabled":                               true,
		"standardized_nudges_misinfo":                                             true,
		"longform_notetweets_rich_text_read_enabled":                              true,
		"responsive_web_enhance_cards_enabled":                                    false,
		"subscriptions_verification_info_enabled":                                 true,
		"subscriptions_verification_info_reason_enabled":                          true,
		"subscriptions_verification_info_verified_since_enabled":                  true,
		"super_follow_badge_privacy_enabled":                                      false,
		"super_follow_exclusive_tweet_notifications_enabled":                      false,
		"super_follow_tweet_api_enabled":                                          false,
		"super_follow_user_api_enabled":                                           false,
		"android_graphql_skip_api_media_color_palette":                            false,
		"creator_subscriptions_subscription_count_enabled":                        false,
		"blue_business_profile_image_shape_enabled":                               false,
		"unified_cards_ad_metadata_container_dynamic_card_content_query_enabled":  false,
		"rweb_video_timestamps_enabled":                                           false,
		"c9s_tweet_anatomy_moderator_badge_enabled":                               false,
		"responsive_web_twitter_article_tweet_consumption_enabled":                false,
	}

	fieldToggles := map[string]interface{}{}

	// 将 features 和 fieldToggles 组合成一个更大的 map
	data := map[string]interface{}{
		"features":     features,
		"fieldToggles": fieldToggles,
		"variables":    variables,
	}
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	req, err := s.newRequestWithData("POST", createTweet, jsonBytes)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 11; Nokia G20) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.88 Mobile Safari/537.36")
	req.Header.Set("x-twitter-auth-type", "OAuth2Client")
	req.Header.Set("x-twitter-active-user", "yes")
	req.Header.Set("x-twitter-client-language", "en")
	err = s.RequestAPI(req, nil)
	if err != nil {
		return "", err
	}
	return "", nil
}
