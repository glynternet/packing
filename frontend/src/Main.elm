module Main exposing (..)

import Browser
import Html exposing (Html, button, div, h3, h4, p, text)
import Html.Events exposing (onClick)
import Http
import Json.Decode
import Json.Encode
import Set
import State



-- MAIN


main =
    Browser.document
        { init = init
        , update = update
        , view = view
        , subscriptions = \_ -> Sub.none
        }



-- MODEL


type alias Flags =
    { state : Maybe String }


type alias StoredState =
    { done : List String }



--TODO: saved state should not be exactly the model. Here there's an issue where fetch results and fetch error can exist at the same time but they should be mutually exclusive.


type alias Model =
    { fetchResults : Maybe (List Group)
    , viewMode : ViewMode
    , error : Maybe String
    , done : Set.Set String
    , showGroups : Bool
    , itemsOnly : Int
    }


type ViewMode
    = ToDo
    | Done


plzResult : Result x x -> x
plzResult res =
    case res of
        Ok ok ->
            ok

        Err err ->
            err


defaultModel : Model
defaultModel =
    Model Nothing ToDo Nothing Set.empty False 0


init : Flags -> ( Model, Cmd Msg )
init flags =
    ( flags.state
        --TODO: catch this error
        |> Maybe.map
            (Json.Decode.decodeString (Json.Decode.field "done" (Json.Decode.nullable (Json.Decode.list Json.Decode.string)))
                >> Result.mapError (Json.Decode.errorToString >> (\errStr -> { defaultModel | error = Just ("Init decode error: " ++ errStr) }))
                >> Result.map (\done -> { defaultModel | done = done |> Maybe.map Set.fromList |> Maybe.withDefault Set.empty })
                >> plzResult
            )
        |> Maybe.withDefault defaultModel
    , Cmd.none
    )



--( flags |> Maybe.withDefault (Model Nothing Nothing Nothing), Cmd.none )
-- UPDATE


type Msg
    = Fetch
    | FetchedResults (Result String (List Group))
    | ViewMode ViewMode
    | ItemDone String Bool
    | ShowGroups Bool
    | ItemsOnly Int
    | ClearDone


contentsDef =
    { groupKeys =
        [ "battery_pack"
        , "board_games"
        , "camera"
        , "clothing"
        , "clothing_bottoms"
        , "clothing_cold"
        , "clothing_general"
        , "clothing_gym"
        , "clothing_hot"
        , "clothing_shoes"
        , "clothing_sunny"
        , "clothing_tops"
        , "clothing_underwear"
        , "clothing_wet"
        , "cycling_bike"
        , "cycling_clothing"
        , "cycling_clothing_cold"
        , "cycling_clothing_essential"
        , "cycling_clothing_mild"
        , "cycling_fluids"
        , "cycling_food"
        , "cycling_garmin"
        , "cycling_guest_bike"
        , "cycling_lights"
        , "cycling_lock"
        , "cycling_tools_ride"
        , "cycling_tools_workshop_portable"
        , "earplugs"
        , "flight"
        , "hiking"
        , "hiking_boots_socks"
        , "hygene_essentials"
        , "hygene_teeth_essentials"
        , "hygene_teeth_medium_or_longtrip"
        , "keyboard_mouse"
        , "keys_phone_wallet"
        , "laptop"
        , "music_player"
        , "outdoors"
        , "phone"
        , "phone_accessories"
        , "phone_and_accessories"
        , "remote_workstation"
        , "smart_watch"
        , "sun"
        , "sunglasses"
        , "sunscreen"
        , "swimming_shorts"
        , "towel"
        , "travel_documents"
        , "travel_utils"
        , "water_bottle"
        , "work_remotely_essentials"
        ]
    , items =
        [ "Shave before going"
        , "Change cassette before going"

        -- Add this to some group
        , "Power meter medals"

        -- Add this to Bay Area location
        , "Bart card"
        ]
    }


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        Fetch ->
            ( model, fetch contentsDef )

        ViewMode mode ->
            ( { model | viewMode = mode }, Cmd.none )

        FetchedResults res ->
            State.updateModel serialiseStateForStorage
                (case res of
                    Ok groups ->
                        { model | error = Nothing, fetchResults = Just groups }

                    Err err ->
                        { model | error = Just err, fetchResults = Nothing }
                )

        ItemDone item done ->
            State.updateModel serialiseStateForStorage
                { model
                    | done =
                        if done then
                            Set.insert item model.done

                        else
                            Set.remove item model.done
                }

        ShowGroups show ->
            ( { model | showGroups = show }, Cmd.none )

        ItemsOnly itemsOnly ->
            ( { model | itemsOnly = itemsOnly }, Cmd.none )

        ClearDone ->
            State.updateModel serialiseStateForStorage { model | done = Set.empty }


serialiseStateForStorage : Model -> String
serialiseStateForStorage model =
    Json.Encode.object
        [ --[ ( "fetchResults", model.fetchResults |> Maybe.map encodeGroups |> Maybe.withDefault Json.Encode.null )
          ( "done", model.done |> (Set.toList >> Json.Encode.list Json.Encode.string) )
        ]
        |> Json.Encode.encode 2


fetch : ContentsDefinition -> Cmd Msg
fetch def =
    defaultPostJSON "/groups/"
        (encodeContentsDefinition def)
        (Http.expectJson (resultFromHttpResult >> FetchedResults) decodeGroups)



-- VIEW


view : Model -> Browser.Document Msg
view model =
    { title = "Packing"
    , body =
        [ div []
            [ button [ onClick Fetch ] [ text "fetch" ]
            , case model.viewMode of
                ToDo ->
                    button [ onClick <| ViewMode Done ] [ text "view done" ]

                Done ->
                    button [ onClick <| ViewMode ToDo ] [ text "view todo" ]
            , button [ onClick <| ShowGroups (not model.showGroups) ] [ text "toggle groups" ]
            , button [ onClick <| ItemsOnly (remainderBy 3 (model.itemsOnly + 1)) ] [ text "toggle items only" ]
            , button [ onClick <| ClearDone ] [ text "reset" ]
            , div []
                (case model.error of
                    Just err ->
                        [ text err ]

                    Nothing ->
                        case model.fetchResults of
                            Nothing ->
                                [ text "Need to fetch" ]

                            Just groups ->
                                h3 []
                                    [ text
                                        ("Viewing "
                                            ++ (case model.viewMode of
                                                    ToDo ->
                                                        "to do"

                                                    Done ->
                                                        "done"
                                               )
                                        )
                                    ]
                                    :: (groups
                                            |> List.sortBy .name
                                            |> List.map
                                                (\group ->
                                                    div []
                                                        (let
                                                            items =
                                                                group.contents.items
                                                                    |> List.filter
                                                                        (\item ->
                                                                            Set.member item model.done
                                                                                |> (case model.viewMode of
                                                                                        ToDo ->
                                                                                            not

                                                                                        Done ->
                                                                                            identity
                                                                                   )
                                                                        )

                                                            toClickableItem itemKey itemText =
                                                                p
                                                                    [ Html.Events.onClick
                                                                        (ItemDone itemKey
                                                                            (case model.viewMode of
                                                                                ToDo ->
                                                                                    True

                                                                                Done ->
                                                                                    False
                                                                            )
                                                                        )
                                                                    ]
                                                                    [ text itemText ]
                                                         in
                                                         if model.itemsOnly > 0 then
                                                            items
                                                                |> List.map
                                                                    (\itemKey ->
                                                                        toClickableItem itemKey
                                                                            ((if model.itemsOnly == 1 then
                                                                                group.name ++ ":"

                                                                              else
                                                                                ""
                                                                             )
                                                                                ++ itemKey
                                                                            )
                                                                    )

                                                         else
                                                            let
                                                                groupDisplayContents =
                                                                    (if List.isEmpty group.contents.groupKeys || not model.showGroups then
                                                                        []

                                                                     else
                                                                        [ h4 [] [ text "groups" ] ] ++ (group.contents.groupKeys |> List.map (\key -> p [] [ text key ]))
                                                                    )
                                                                        ++ (if List.isEmpty items then
                                                                                []

                                                                            else
                                                                                [ h4 [] [ text "items" ] ]
                                                                                    ++ (items
                                                                                            |> List.map (\key -> toClickableItem key key)
                                                                                       )
                                                                           )
                                                            in
                                                            if List.isEmpty groupDisplayContents then
                                                                []

                                                            else
                                                                List.concat [ [ h3 [] [ text group.name ] ], groupDisplayContents ]
                                                        )
                                                )
                                       )
                )
            ]
        ]
    }



--- HTTP


type alias Group =
    { name : String, contents : ContentsDefinition }


type alias ContentsDefinition =
    { groupKeys : List String, items : List String }


encodeGroups : List Group -> Json.Encode.Value
encodeGroups =
    Json.Encode.list
        (\group ->
            Json.Encode.object
                [ ( "name", Json.Encode.string group.name )
                , ( "contents", encodeContentsDefinition group.contents )
                ]
        )


encodeContentsDefinition : ContentsDefinition -> Json.Encode.Value
encodeContentsDefinition def =
    Json.Encode.object
        [ ( "group_keys", Json.Encode.list Json.Encode.string def.groupKeys )
        , ( "items", Json.Encode.list Json.Encode.string def.items )
        ]


decodeGroups : Json.Decode.Decoder (List Group)
decodeGroups =
    Json.Decode.list
        (Json.Decode.map2 Group
            (Json.Decode.field "name" Json.Decode.string)
            (Json.Decode.field "contents"
                (Json.Decode.map2 ContentsDefinition
                    (Json.Decode.field "group_keys" <| decodedWithNullAsDefault [] <| Json.Decode.list Json.Decode.string)
                    (Json.Decode.field "items" <| decodedWithNullAsDefault [] <| Json.Decode.list Json.Decode.string)
                )
            )
        )


decodedWithNullAsDefault : a -> Json.Decode.Decoder a -> Json.Decode.Decoder a
decodedWithNullAsDefault default decoder =
    Json.Decode.map (Maybe.withDefault default) (Json.Decode.nullable decoder)


defaultPostJSON : String -> Json.Encode.Value -> Http.Expect msg -> Cmd msg
defaultPostJSON url jsonValue expect =
    Http.request
        { method = "POST"
        , headers = []
        , url = url
        , body = Http.jsonBody jsonValue
        , expect = expect
        , timeout = Just 2000
        , tracker = Nothing
        }


resultFromHttpResult : Result Http.Error a -> Result String a
resultFromHttpResult =
    Result.mapError dataReceivedErrToString


dataReceivedErrToString : Http.Error -> String
dataReceivedErrToString error =
    case error of
        Http.BadUrl url ->
            "The URL " ++ url ++ " was invalid"

        Http.Timeout ->
            "Unable to reach the server, try again"

        Http.NetworkError ->
            "Unable to reach the server, check your network connection"

        Http.BadStatus code ->
            "Unable to get data. Status: " ++ String.fromInt code

        Http.BadBody errorMessage ->
            "Data received was not in the correct format. Error message: " ++ errorMessage
