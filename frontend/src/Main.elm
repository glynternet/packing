module Main exposing (..)

import Browser
import Html exposing (Html, a, button, div, h3, h4, p, text, textarea)
import Html.Attributes exposing (href, id, rows, style, value)
import Html.Events exposing (onClick, onInput)
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
    { done : List String
    , selection : Maybe String
    }


type alias Model =
    { fetchResults : Maybe (List Group)
    , viewMode : ViewMode
    , error : Maybe String
    , done : Set.Set String

    -- showGroupLinks mirrors the CLI's --include-group-references: show the links
    -- to each group's nested groups.
    , showGroupLinks : Bool

    -- showContainerGroups mirrors the CLI's --include-empty-parent-groups: show
    -- groups that only bundle other groups and have no items of their own.
    , showContainerGroups : Bool
    , itemsOnly : Int
    , selectionText : String

    -- requestId tags each in-flight fetch so out-of-order responses (e.g. while
    -- typing quickly) can be discarded; only the latest request's result is used.
    , requestId : Int
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


{-| The selection a user starts with, in the same text format as a selection
file: ref:/req: tagged lines, plain lines as individual items, # comments.
-}
defaultSelectionText : String
defaultSelectionText =
    """ref: battery_pack
ref: board_games
ref: camera
ref: clothing
ref: clothing_bottoms
ref: clothing_cold
ref: clothing_general
ref: clothing_gym
ref: clothing_hot
ref: clothing_shoes
ref: clothing_sunny
ref: clothing_tops
ref: clothing_underwear
ref: clothing_wet
ref: cycling_bike
ref: cycling_clothing
ref: cycling_clothing_cold
ref: cycling_clothing_essential
ref: cycling_clothing_mild
ref: cycling_fluids
ref: cycling_food
ref: cycling_garmin
ref: cycling_guest_bike
ref: cycling_lights
ref: cycling_lock
ref: cycling_tools_ride
ref: cycling_tools_workshop_portable
ref: earplugs
ref: flight
ref: hiking
ref: hiking_boots_socks
ref: hygene_essentials
ref: hygene_teeth_essentials
ref: hygene_teeth_medium_or_longtrip
ref: keyboard_mouse
ref: keys_phone_wallet
ref: laptop
ref: music_player
ref: outdoors
ref: phone
ref: phone_accessories
ref: phone_and_accessories
ref: remote_workstation
ref: smart_watch
ref: sun
ref: sunglasses
ref: sunscreen
ref: swimming_shorts
ref: towel
ref: travel_documents
ref: travel_utils
ref: water_bottle
ref: work_remotely_essentials

Shave before going
Change cassette before going

# Add this to some group
Power meter medals

# Add this to Bay Area location
Bart card
"""


defaultModel : Model
defaultModel =
    { fetchResults = Nothing
    , viewMode = ToDo
    , error = Nothing
    , done = Set.empty
    , showGroupLinks = False
    , showContainerGroups = False
    , itemsOnly = 0
    , selectionText = defaultSelectionText
    , requestId = 0
    }


init : Flags -> ( Model, Cmd Msg )
init flags =
    let
        loaded =
            flags.state
                |> Maybe.map
                    (Json.Decode.decodeString storedStateDecoder
                        >> Result.mapError (\err -> { defaultModel | error = Just ("Init decode error: " ++ Json.Decode.errorToString err) })
                        >> Result.map
                            (\stored ->
                                { defaultModel
                                    | done = Set.fromList stored.done
                                    , selectionText = stored.selection |> Maybe.withDefault defaultSelectionText
                                }
                            )
                        >> plzResult
                    )
                |> Maybe.withDefault defaultModel

        firstId =
            loaded.requestId + 1
    in
    ( { loaded | requestId = firstId }
    , fetch firstId loaded.selectionText
    )



-- UPDATE


type Msg
    = FetchedResults Int (Result String (List Group))
    | ViewMode ViewMode
    | ItemDone String Bool
    | ShowGroupLinks Bool
    | ShowContainerGroups Bool
    | ItemsOnly Int
    | ClearDone
    | SelectionChanged String


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        SelectionChanged selectionText ->
            let
                newId =
                    model.requestId + 1

                newModel =
                    { model | selectionText = selectionText, requestId = newId }
            in
            ( newModel
            , Cmd.batch
                [ State.storeState (serialiseStateForStorage newModel)
                , fetch newId selectionText
                ]
            )

        FetchedResults id res ->
            if id /= model.requestId then
                -- Stale response for a selection that has since changed; ignore it.
                ( model, Cmd.none )

            else
                ( case res of
                    Ok groups ->
                        { model | error = Nothing, fetchResults = Just groups }

                    Err err ->
                        -- Keep the last good render visible while showing the error.
                        { model | error = Just err }
                , Cmd.none
                )

        ViewMode mode ->
            ( { model | viewMode = mode }, Cmd.none )

        ItemDone item done ->
            State.updateModel serialiseStateForStorage
                { model
                    | done =
                        if done then
                            Set.insert item model.done

                        else
                            Set.remove item model.done
                }

        ShowGroupLinks show ->
            ( { model | showGroupLinks = show }, Cmd.none )

        ShowContainerGroups show ->
            ( { model | showContainerGroups = show }, Cmd.none )

        ItemsOnly itemsOnly ->
            ( { model | itemsOnly = itemsOnly }, Cmd.none )

        ClearDone ->
            State.updateModel serialiseStateForStorage { model | done = Set.empty }


serialiseStateForStorage : Model -> String
serialiseStateForStorage model =
    Json.Encode.object
        [ ( "done", model.done |> (Set.toList >> Json.Encode.list Json.Encode.string) )
        , ( "selection", Json.Encode.string model.selectionText )
        ]
        |> Json.Encode.encode 2


storedStateDecoder : Json.Decode.Decoder StoredState
storedStateDecoder =
    Json.Decode.map2 StoredState
        (Json.Decode.field "done" (decodedWithNullAsDefault [] (Json.Decode.list Json.Decode.string)))
        (Json.Decode.maybe (Json.Decode.field "selection" Json.Decode.string))


fetch : Int -> String -> Cmd Msg
fetch id selectionText =
    Http.request
        { method = "POST"
        , headers = []
        , url = "/selection/"
        , body = Http.stringBody "text/plain" selectionText
        , expect = expectGroups (FetchedResults id)
        , timeout = Just 5000
        , tracker = Nothing
        }



-- VIEW


view : Model -> Browser.Document Msg
view model =
    { title = "Packing"
    , body =
        [ div
            [ style "display" "flex"
            , style "gap" "1rem"
            , style "align-items" "flex-start"
            , style "padding" "1rem"
            ]
            [ div [ style "flex" "1 1 0" ]
                [ h3 [] [ text "Selection" ]
                , p [] [ text "Edit your selection below. Copy the text out to save it." ]
                , textarea
                    [ value model.selectionText
                    , onInput SelectionChanged
                    , rows 30
                    , style "width" "100%"
                    , style "box-sizing" "border-box"
                    , style "font-family" "monospace"
                    ]
                    []
                ]
            , div [ style "flex" "1 1 0" ]
                [ controlsView model
                , resultsView model
                ]
            ]
        ]
    }


controlsView : Model -> Html Msg
controlsView model =
    div []
        [ case model.viewMode of
            ToDo ->
                button [ onClick <| ViewMode Done ] [ text "view done" ]

            Done ->
                button [ onClick <| ViewMode ToDo ] [ text "view todo" ]
        , button [ onClick <| ShowGroupLinks (not model.showGroupLinks) ] [ text "show group links" ]
        , button [ onClick <| ShowContainerGroups (not model.showContainerGroups) ] [ text "show container groups" ]
        , button [ onClick <| ItemsOnly (remainderBy 3 (model.itemsOnly + 1)) ] [ text "toggle items only" ]
        , button [ onClick <| ClearDone ] [ text "reset" ]
        ]


resultsView : Model -> Html Msg
resultsView model =
    div []
        (List.concat
            [ case model.error of
                Just err ->
                    [ p [ style "color" "red" ] [ text err ] ]

                Nothing ->
                    []
            , case model.fetchResults of
                Nothing ->
                    [ text "Editing selection…" ]

                Just groups ->
                    groupsView model groups
            ]
        )


groupsView : Model -> List Group -> List (Html Msg)
groupsView model groups =
    let
        -- Reverse of contents.refs: the groups that directly reference `name`.
        parentsOf name =
            groups
                |> List.filter (\g -> List.member name g.contents.refs)
                |> List.map .name
                |> List.sort
    in
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
                        -- id is the group name so nested-group refs can link to it (href="#name")
                        div [ id group.name ]
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
                                        [ style "cursor" "pointer"
                                        , Html.Events.onClick
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
                                    -- Back-links to the parent groups that contain this group, if any.
                                    parentsLine =
                                        case parentsOf group.name of
                                            [] ->
                                                []

                                            parents ->
                                                [ p [ style "font-size" "0.85em", style "color" "#666" ]
                                                    (text "part of: "
                                                        :: (parents
                                                                |> List.map (\parent -> a [ href ("#" ++ parent), style "cursor" "pointer" ] [ text parent ])
                                                                |> List.intersperse (text " · ")
                                                           )
                                                    )
                                                ]

                                    -- A container (empty-parent) group bundles other groups but has no items of its own.
                                    isContainer =
                                        List.isEmpty group.contents.items

                                    groupDisplayContents =
                                        (if model.showGroupLinks && not (List.isEmpty group.contents.refs) then
                                            [ h4 [] [ text "groups" ] ] ++ (group.contents.refs |> List.map (\key -> p [] [ a [ href ("#" ++ key), style "cursor" "pointer" ] [ text key ] ]))

                                         else
                                            []
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
                                if isContainer && not model.showContainerGroups then
                                    -- Empty-parent group: hidden unless the container-groups toggle is on.
                                    []

                                else if not isContainer && List.isEmpty groupDisplayContents then
                                    -- Ordinary group with nothing left to show (e.g. all its items are done).
                                    []

                                else
                                    List.concat [ [ h3 [] [ text group.name ] ], parentsLine, groupDisplayContents ]
                            )
                    )
           )



--- HTTP


type alias Group =
    { name : String, contents : ContentsDefinition }


type alias ContentsDefinition =
    { refs : List String, items : List String }


decodeGroups : Json.Decode.Decoder (List Group)
decodeGroups =
    Json.Decode.list
        (Json.Decode.map2 Group
            (Json.Decode.field "name" Json.Decode.string)
            (Json.Decode.field "contents"
                (Json.Decode.map2 ContentsDefinition
                    (Json.Decode.field "refs" <| decodedWithNullAsDefault [] <| Json.Decode.list Json.Decode.string)
                    (Json.Decode.field "items" <| decodedWithNullAsDefault [] <| Json.Decode.list Json.Decode.string)
                )
            )
        )


decodedWithNullAsDefault : a -> Json.Decode.Decoder a -> Json.Decode.Decoder a
decodedWithNullAsDefault default decoder =
    Json.Decode.map (Maybe.withDefault default) (Json.Decode.nullable decoder)


{-| expectGroups decodes a successful JSON group response, and on a non-2xx
status surfaces the server's plain-text error body (e.g. a selection parse
error) so it can be shown to the user directly.
-}
expectGroups : (Result String (List Group) -> msg) -> Http.Expect msg
expectGroups toMsg =
    Http.expectStringResponse toMsg <|
        \response ->
            case response of
                Http.BadUrl_ url ->
                    Err ("The URL " ++ url ++ " was invalid")

                Http.Timeout_ ->
                    Err "Unable to reach the server, try again"

                Http.NetworkError_ ->
                    Err "Unable to reach the server, check your network connection"

                Http.BadStatus_ metadata body ->
                    Err
                        (if String.isEmpty (String.trim body) then
                            "Server error, status: " ++ String.fromInt metadata.statusCode

                         else
                            String.trim body
                        )

                Http.GoodStatus_ _ body ->
                    Json.Decode.decodeString decodeGroups body
                        |> Result.mapError
                            (\err -> "Data received was not in the correct format: " ++ Json.Decode.errorToString err)
