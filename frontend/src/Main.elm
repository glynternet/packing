module Main exposing (..)

import Browser
import Browser.Events
import Dict exposing (Dict)
import Html exposing (Html, a, button, div, h3, h4, p, span, text, textarea)
import Html.Attributes exposing (href, id, rows, style, value)
import Html.Events exposing (onClick, onInput)
import Http
import Json.Decode
import Json.Encode
import Set
import State
import Svg
import Svg.Attributes as SA



-- MAIN


main =
    Browser.document
        { init = init
        , update = update
        , view = view
        , subscriptions = subscriptions
        }



-- MODEL


type alias Flags =
    { state : Maybe String }


type alias StoredState =
    { done : List String
    , selection : Maybe String
    , doneDisplay : Maybe String
    }


type alias Model =
    { fetchResults : Maybe (List Group)
    , viewMode : ViewMode
    , done : Set.Set String

    -- showGroupLinks mirrors the CLI's --include-group-references: show the links
    -- to each group's nested groups.
    , showGroupLinks : Bool

    -- showContainerGroups mirrors the CLI's --include-empty-parent-groups: show
    -- groups that only bundle other groups and have no items of their own.
    , showContainerGroups : Bool

    -- inlineSingletons folds "singleton" groups — whose entire content is a
    -- single member (one item, or one ref forming a passthrough chain) — into
    -- whatever references them, instead of showing a standalone group. Drives
    -- both the list and the graph.
    , inlineSingletons : Bool

    -- doneDisplay decides what the list view does with done items and groups:
    -- Hide takes them off the page, which is what makes the ToDo/Done split
    -- below meaningful; Mark keeps them on a single combined list — faded and
    -- sorted last — and the ToDo/Done distinction is ignored.
    , doneDisplay : DoneDisplay
    , itemsOnly : Int
    , selectionText : String

    -- simplifyNote holds the outcome of the most recent Simplify action (the
    -- refs/items that were removed), shown next to the editor until the user
    -- edits the selection again. Nothing when no Simplify has run since.
    , simplifyNote : Maybe { refs : List String, items : List String }

    -- requestId tags each in-flight fetch so out-of-order responses (e.g. while
    -- typing quickly) can be discarded; only the latest request's result is used.
    , requestId : Int

    -- renderStatus reflects the outcome of the most recent /selection/ request,
    -- surfaced as a marker next to the Selection header.
    , renderStatus : RenderStatus

    -- Graph-view viewport: pan/zoom of the SVG group graph, the in-flight pan
    -- drag (if any), and the node the user has clicked to focus on.
    , graphScale : Float
    , graphPanX : Float
    , graphPanY : Float
    , graphDrag : Maybe { startX : Float, startY : Float, lastX : Float, lastY : Float }
    , graphSelected : Maybe String
    }


type ViewMode
    = ToDo
    | Done
    | Graph


{-| How the list view treats items and groups that are done.
-}
type DoneDisplay
    = -- done items and groups leave the page; the ToDo/Done pages apply
      Hide
      -- done items and groups stay on the page, faded and sorted last
    | Mark


{-| The stored-state spelling of a DoneDisplay, and back. Kept as a pair so the
encoder and decoder cannot drift; an unrecognised (or absent) value reads as the
default, Mark.
-}
doneDisplayKey : DoneDisplay -> String
doneDisplayKey display =
    case display of
        Hide ->
            "hidden"

        Mark ->
            "marked"


doneDisplayFromKey : Maybe String -> DoneDisplay
doneDisplayFromKey key =
    if key == Just (doneDisplayKey Hide) then
        Hide

    else
        Mark


type RenderStatus
    = Rendering
    | Rendered
    | RenderFailed String


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
    , done = Set.empty
    , showGroupLinks = False
    , showContainerGroups = False
    , inlineSingletons = True
    , doneDisplay = Mark
    , itemsOnly = 0
    , selectionText = defaultSelectionText
    , simplifyNote = Nothing
    , requestId = 0
    , renderStatus = Rendering
    , graphScale = 1
    , graphPanX = 20
    , graphPanY = 20
    , graphDrag = Nothing
    , graphSelected = Nothing
    }


init : Flags -> ( Model, Cmd Msg )
init flags =
    let
        loaded =
            flags.state
                |> Maybe.map
                    (Json.Decode.decodeString storedStateDecoder
                        -- On corrupt stored state, fall back to defaults.
                        >> Result.mapError (\_ -> defaultModel)
                        >> Result.map
                            (\stored ->
                                { defaultModel
                                    | done = Set.fromList stored.done
                                    , selectionText = stored.selection |> Maybe.withDefault defaultSelectionText
                                    , doneDisplay = doneDisplayFromKey stored.doneDisplay
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
    | ToggleInlineSingletons Bool
    | SetDoneDisplay DoneDisplay
    | ItemsOnly Int
    | ClearDone
    | SelectionChanged String
    | SimplifyClicked
    | Simplified (Result String SimplifyResponse)
    | GraphWheel Float Float Float
    | GraphDragStart Float Float
    | GraphDragMove Float Float
    | GraphDragEnd Float Float
    | GraphNodeClicked String
    | GraphZoom Float
    | GraphFit
    | ClearGraphFocus


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        SelectionChanged selectionText ->
            let
                newId =
                    model.requestId + 1

                newModel =
                    { model | selectionText = selectionText, simplifyNote = Nothing, requestId = newId, renderStatus = Rendering }
            in
            ( newModel
            , Cmd.batch
                [ State.storeState (serialiseStateForStorage newModel)
                , fetch newId selectionText
                ]
            )

        SimplifyClicked ->
            -- Fire-and-forget: the response (Simplified) applies the minified
            -- selection and re-renders. Leave the model as-is meanwhile.
            ( model, simplifyRequest model.selectionText )

        Simplified result ->
            case result of
                Ok resp ->
                    let
                        newId =
                            model.requestId + 1

                        newModel =
                            { model
                                | selectionText = resp.selection
                                , simplifyNote = Just { refs = resp.removedRefs, items = resp.removedItems }
                                , requestId = newId
                                , renderStatus = Rendering
                            }
                    in
                    ( newModel
                    , Cmd.batch
                        [ State.storeState (serialiseStateForStorage newModel)
                        , fetch newId resp.selection
                        ]
                    )

                Err err ->
                    -- Keep the current selection; surface the error via the badge.
                    ( { model | renderStatus = RenderFailed err }, Cmd.none )

        FetchedResults id res ->
            if id /= model.requestId then
                -- Stale response for a selection that has since changed; ignore it.
                ( model, Cmd.none )

            else
                ( case res of
                    Ok groups ->
                        { model | fetchResults = Just groups, renderStatus = Rendered }

                    Err err ->
                        -- Keep the last good render visible; the marker shows the error.
                        { model | renderStatus = RenderFailed err }
                , Cmd.none
                )

        ViewMode Graph ->
            -- Entering the graph view: frame the whole graph to fit the viewport.
            let
                fit =
                    fitToView (effectiveResults model)
            in
            ( { model
                | viewMode = Graph
                , graphScale = fit.scale
                , graphPanX = fit.panX
                , graphPanY = fit.panY
                , graphSelected = Nothing
                , graphDrag = Nothing
              }
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

        ToggleInlineSingletons enabled ->
            let
                base =
                    { model | inlineSingletons = enabled }
            in
            case model.viewMode of
                Graph ->
                    -- Toggling changes which nodes exist, so refit the viewport.
                    let
                        fit =
                            fitToView (effectiveResults base)
                    in
                    ( { base
                        | graphScale = fit.scale
                        , graphPanX = fit.panX
                        , graphPanY = fit.panY
                        , graphSelected = Nothing
                      }
                    , Cmd.none
                    )

                _ ->
                    ( base, Cmd.none )

        SetDoneDisplay display ->
            -- Mark shows a single combined list, so the Done page is meaningless
            -- there: drop back to ToDo rather than leave the user on it (and
            -- land them back on it when they later switch to Hide).
            State.updateModel serialiseStateForStorage
                { model
                    | doneDisplay = display
                    , viewMode =
                        if display == Mark && model.viewMode == Done then
                            ToDo

                        else
                            model.viewMode
                }

        ItemsOnly itemsOnly ->
            ( { model | itemsOnly = itemsOnly }, Cmd.none )

        ClearDone ->
            State.updateModel serialiseStateForStorage { model | done = Set.empty }

        GraphWheel deltaY offsetX offsetY ->
            -- Zoom towards the cursor: keep the graph point under the pointer fixed.
            let
                factor =
                    if deltaY < 0 then
                        1.1

                    else
                        1 / 1.1

                newScale =
                    clamp 0.1 4 (model.graphScale * factor)

                worldX =
                    (offsetX - model.graphPanX) / model.graphScale

                worldY =
                    (offsetY - model.graphPanY) / model.graphScale
            in
            ( { model
                | graphScale = newScale
                , graphPanX = offsetX - worldX * newScale
                , graphPanY = offsetY - worldY * newScale
              }
            , Cmd.none
            )

        GraphDragStart x y ->
            -- Grabbing empty canvas starts a pan. The focus is only cleared on a
            -- click with no movement (see GraphDragEnd), so panning keeps it.
            ( { model | graphDrag = Just { startX = x, startY = y, lastX = x, lastY = y } }, Cmd.none )

        GraphDragMove x y ->
            case model.graphDrag of
                Just d ->
                    ( { model
                        | graphPanX = model.graphPanX + (x - d.lastX)
                        , graphPanY = model.graphPanY + (y - d.lastY)
                        , graphDrag = Just { d | lastX = x, lastY = y }
                      }
                    , Cmd.none
                    )

                Nothing ->
                    ( model, Cmd.none )

        GraphDragEnd x y ->
            -- A background press that didn't move is a click: clear the focus and
            -- refit the full graph. A press that moved was a pan: keep the focus.
            let
                wasClick =
                    case model.graphDrag of
                        Just d ->
                            abs (x - d.startX) + abs (y - d.startY) < 4

                        Nothing ->
                            False
            in
            if wasClick && model.graphSelected /= Nothing then
                let
                    fit =
                        fitToView (effectiveResults model)
                in
                ( { model
                    | graphDrag = Nothing
                    , graphSelected = Nothing
                    , graphScale = fit.scale
                    , graphPanX = fit.panX
                    , graphPanY = fit.panY
                  }
                , Cmd.none
                )

            else
                ( { model | graphDrag = Nothing }, Cmd.none )

        GraphNodeClicked name ->
            -- Toggle focus: clicking the focused node again clears the focus.
            -- Selecting refits to that node's lineage; deselecting refits full.
            let
                newSelected =
                    if model.graphSelected == Just name then
                        Nothing

                    else
                        Just name

                viewGroups =
                    case newSelected of
                        Just sel ->
                            effectiveResults model |> Maybe.map (lineageGroups sel)

                        Nothing ->
                            effectiveResults model

                fit =
                    fitToView viewGroups
            in
            ( { model
                | graphSelected = newSelected
                , graphDrag = Nothing
                , graphScale = fit.scale
                , graphPanX = fit.panX
                , graphPanY = fit.panY
              }
            , Cmd.none
            )

        GraphZoom factor ->
            -- Button zoom, centred on a fixed point near the middle of the canvas.
            let
                newScale =
                    clamp 0.1 4 (model.graphScale * factor)

                cx =
                    450

                cy =
                    280

                worldX =
                    (cx - model.graphPanX) / model.graphScale

                worldY =
                    (cy - model.graphPanY) / model.graphScale
            in
            ( { model
                | graphScale = newScale
                , graphPanX = cx - worldX * newScale
                , graphPanY = cy - worldY * newScale
              }
            , Cmd.none
            )

        GraphFit ->
            let
                fit =
                    fitToView (effectiveResults model)
            in
            ( { model | graphScale = fit.scale, graphPanX = fit.panX, graphPanY = fit.panY, graphSelected = Nothing }, Cmd.none )

        ClearGraphFocus ->
            -- Reset the lineage filter and refit the full graph.
            let
                fit =
                    fitToView (effectiveResults model)
            in
            ( { model | graphScale = fit.scale, graphPanX = fit.panX, graphPanY = fit.panY, graphSelected = Nothing }, Cmd.none )


serialiseStateForStorage : Model -> String
serialiseStateForStorage model =
    Json.Encode.object
        [ ( "done", model.done |> (Set.toList >> Json.Encode.list Json.Encode.string) )
        , ( "selection", Json.Encode.string model.selectionText )
        , ( "doneDisplay", Json.Encode.string (doneDisplayKey model.doneDisplay) )
        ]
        |> Json.Encode.encode 2


storedStateDecoder : Json.Decode.Decoder StoredState
storedStateDecoder =
    Json.Decode.map3 StoredState
        (Json.Decode.field "done" (decodedWithNullAsDefault [] (Json.Decode.list Json.Decode.string)))
        (Json.Decode.maybe (Json.Decode.field "selection" Json.Decode.string))
        (Json.Decode.maybe (Json.Decode.field "doneDisplay" Json.Decode.string))


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


simplifyRequest : String -> Cmd Msg
simplifyRequest selectionText =
    Http.request
        { method = "POST"
        , headers = []
        , url = "/simplify/"
        , body = Http.stringBody "text/plain" selectionText
        , expect = expectSimplify Simplified
        , timeout = Just 5000
        , tracker = Nothing
        }



-- SUBSCRIPTIONS


{-| While a graph pan is in progress, track the mouse globally so panning
continues even if the pointer leaves the SVG, and ends on mouse-up.
-}
subscriptions : Model -> Sub Msg
subscriptions model =
    case model.graphDrag of
        Just _ ->
            Sub.batch
                [ Browser.Events.onMouseMove
                    (Json.Decode.map2 GraphDragMove
                        (Json.Decode.field "clientX" Json.Decode.float)
                        (Json.Decode.field "clientY" Json.Decode.float)
                    )
                , Browser.Events.onMouseUp
                    (Json.Decode.map2 GraphDragEnd
                        (Json.Decode.field "clientX" Json.Decode.float)
                        (Json.Decode.field "clientY" Json.Decode.float)
                    )
                ]

        Nothing ->
            Sub.none



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
                [ h3 [] [ text "Selection ", renderStatusBadge model.renderStatus ]
                , renderStatusMessage model.renderStatus
                , p [] [ text "Edit your selection below. Copy the text out to save it." ]
                , div [ style "margin-bottom" "0.5rem" ]
                    [ button [ onClick SimplifyClicked ] [ text "Simplify" ]
                    , span [ style "margin-left" "0.5rem", style "color" "#555", style "font-size" "0.85rem" ]
                        [ text "remove refs already pulled in by others, and items already in a group" ]
                    ]
                , simplifyNoteView model.simplifyNote
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


{-| A small marker shown next to the "Selection" header reflecting the outcome
of the most recent /selection/ request.
-}
renderStatusBadge : RenderStatus -> Html msg
renderStatusBadge status =
    case status of
        Rendering ->
            span [ Html.Attributes.title "Rendering…" ] [ text "⏳" ]

        Rendered ->
            span [ Html.Attributes.title "Rendered successfully" ] [ text "✅" ]

        RenderFailed _ ->
            span [ Html.Attributes.title "Render failed", style "color" "red" ] [ text "❌" ]


{-| On failure, the render error message shown under the header; nothing otherwise.
-}
renderStatusMessage : RenderStatus -> Html msg
renderStatusMessage status =
    case status of
        RenderFailed err ->
            p [ style "color" "red" ] [ text err ]

        _ ->
            text ""


{-| The result of the most recent Simplify: which refs/items were removed, or a
"nothing to remove" note. Cleared (Nothing) once the user edits the selection.
-}
simplifyNoteView : Maybe { refs : List String, items : List String } -> Html Msg
simplifyNoteView note =
    case note of
        Nothing ->
            text ""

        Just { refs, items } ->
            if List.isEmpty refs && List.isEmpty items then
                p [ style "color" "green", style "font-size" "0.85rem" ]
                    [ text "Selection already minimal; nothing removed." ]

            else
                div [ style "color" "#555", style "font-size" "0.85rem", style "margin-bottom" "0.5rem" ]
                    (p [ style "margin" "0 0 0.25rem 0" ] [ text "Simplified — removed:" ]
                        :: removedLine "ref" refs
                        ++ removedLine "item" items
                    )


{-| A single "N refs: a, b, c" line for the simplify note, or nothing when the
list is empty.
-}
removedLine : String -> List String -> List (Html Msg)
removedLine noun xs =
    if List.isEmpty xs then
        []

    else
        [ p [ style "margin" "0 0 0.25rem 0" ]
            [ text (String.fromInt (List.length xs) ++ " " ++ noun ++ "(s): " ++ String.join ", " xs) ]
        ]


{-| Label for the inline-singletons toggle, reflecting the current state
(mirrors the stateful "view done"/"view todo" button).
-}
inlineToggleLabel : Bool -> String
inlineToggleLabel enabled =
    if enabled then
        "singletons: inlined"

    else
        "singletons: grouped"


{-| Label for the done-display toggle, reflecting the current state (mirrors
inlineToggleLabel).
-}
doneDisplayLabel : DoneDisplay -> String
doneDisplayLabel display =
    case display of
        Hide ->
            "done: hidden"

        Mark ->
            "done: marked"


{-| The done display the toggle button switches to.
-}
otherDoneDisplay : DoneDisplay -> DoneDisplay
otherDoneDisplay display =
    case display of
        Hide ->
            Mark

        Mark ->
            Hide


controlsView : Model -> Html Msg
controlsView model =
    case model.viewMode of
        Graph ->
            div []
                [ button [ onClick <| ViewMode ToDo ] [ text "list view" ]
                , button [ onClick <| GraphZoom 1.25 ] [ text "zoom in" ]
                , button [ onClick <| GraphZoom 0.8 ] [ text "zoom out" ]
                , button [ onClick GraphFit ] [ text "fit" ]
                , button [ onClick <| ToggleInlineSingletons (not model.inlineSingletons) ]
                    [ text (inlineToggleLabel model.inlineSingletons) ]
                , span [ style "margin-left" "0.75rem", style "font-size" "0.85em" ]
                    [ legendSwatch "#e2e8f0" "#a0aec0"
                    , text " group  "
                    , legendSwatch "#c6f6d5" "#68d391"
                    , text " item"
                    ]
                , span [ style "margin-left" "0.75rem", style "font-size" "0.85em", style "color" "#666" ]
                    [ text "drag to pan · scroll to zoom · click a node to focus its lineage · click empty space or ✕ to reset" ]
                ]

        _ ->
            div []
                ([ button [ onClick <| SetDoneDisplay (otherDoneDisplay model.doneDisplay) ]
                    [ text (doneDisplayLabel model.doneDisplay) ]
                 ]
                    -- The todo/done pages only exist while done work is hidden;
                    -- Mark puts both on one page.
                    ++ (case model.doneDisplay of
                            Hide ->
                                [ case model.viewMode of
                                    Done ->
                                        button [ onClick <| ViewMode ToDo ] [ text "view todo" ]

                                    _ ->
                                        button [ onClick <| ViewMode Done ] [ text "view done" ]
                                ]

                            Mark ->
                                []
                       )
                    ++ [ button [ onClick <| ViewMode Graph ] [ text "graph view" ]
                       , button [ onClick <| ShowGroupLinks (not model.showGroupLinks) ] [ text "show group links" ]
                       , button [ onClick <| ShowContainerGroups (not model.showContainerGroups) ] [ text "show container groups" ]
                       , button [ onClick <| ToggleInlineSingletons (not model.inlineSingletons) ]
                            [ text (inlineToggleLabel model.inlineSingletons) ]
                       , button [ onClick <| ItemsOnly (remainderBy 3 (model.itemsOnly + 1)) ] [ text "toggle items only" ]
                       , button [ onClick <| ClearDone ] [ text "reset" ]
                       ]
                )


resultsView : Model -> Html Msg
resultsView model =
    -- Errors are surfaced by the render-status marker next to the Selection
    -- header, so this panel only shows the last successfully rendered groups.
    div []
        (case effectiveResults model of
            Nothing ->
                [ text "Editing selection…" ]

            Just groups ->
                case model.viewMode of
                    Graph ->
                        [ graphFocusBanner model.graphSelected, graphView model (graphGroups model groups) ]

                    _ ->
                        groupsView model groups
        )


{-| The rendered groups after applying view transforms that both the list and
graph share. When inlineSingletons is on, singleton groups are folded away:
collapsePassthroughRefs first rewrites single-ref passthrough chains down to
their terminal, then collapseSingleItemGroups inlines single-item groups (so a
chain ending in a single item composes into that item).
-}
effectiveResults : Model -> Maybe (List Group)
effectiveResults model =
    model.fetchResults
        |> Maybe.map
            (\groups ->
                if model.inlineSingletons then
                    groups |> collapsePassthroughRefs |> collapseSingleItemGroups

                else
                    groups
            )


{-| Fold every reference that resolves to a single item into the groups that
reference it: drop the standalone one-item group and append its item to each
referencing group (removing the ref). A one-item group with no parent is left
as-is so its item is never lost. Because the inlined item keeps the item's own
name, its done-state is shared across every group it lands in.

Single-pass: the collapsible set is taken from the input, so a group that only
becomes single-item as a result of another collapsing is not itself collapsed.
-}
collapseSingleItemGroups : List Group -> List Group
collapseSingleItemGroups groups =
    let
        referenced =
            groups |> List.concatMap (.contents >> .refs) |> Set.fromList

        collapsible =
            groups
                |> List.filterMap
                    (\g ->
                        case ( g.contents.refs, g.contents.items ) of
                            ( [], [ only ] ) ->
                                if Set.member g.name referenced then
                                    Just ( g.name, only )

                                else
                                    Nothing

                            _ ->
                                Nothing
                    )
                |> Dict.fromList
    in
    groups
        |> List.filter (\g -> not (Dict.member g.name collapsible))
        |> List.map
            (\g ->
                { g
                    | contents =
                        { refs = g.contents.refs |> List.filter (\r -> not (Dict.member r collapsible))
                        , items = g.contents.items ++ (g.contents.refs |> List.filterMap (\r -> Dict.get r collapsible))
                        }
                }
            )


{-| Collapse chains of "passthrough" groups — a group whose entire content is a
single ref — to their terminal (the first name in the chain that is not itself a
passthrough). Every ref pointing at a passthrough head is rewritten to that
terminal, and the resolvable passthrough groups are dropped. A chain that cycles
is left intact (never rewritten, never dropped) so resolution always terminates.

Runs before collapseSingleItemGroups so the two compose: a chain terminating in
a single-item group gets its refs rewritten to that group, which the single-item
step then inlines. Mirrors Go `pkg/inline.PassthroughRefs`.
-}
collapsePassthroughRefs : List Group -> List Group
collapsePassthroughRefs groups =
    let
        -- name -> its single ref, for every passthrough group.
        passthrough =
            groups
                |> List.filterMap
                    (\g ->
                        case ( g.contents.refs, g.contents.items ) of
                            ( [ only ], [] ) ->
                                Just ( g.name, only )

                            _ ->
                                Nothing
                    )
                |> Dict.fromList

        -- Follow the chain from a passthrough; `Just terminal` on reaching a
        -- non-passthrough, `Nothing` on a cycle.
        walk current visited =
            case Dict.get current passthrough of
                Nothing ->
                    Just current

                Just next ->
                    if Set.member current visited then
                        Nothing

                    else
                        walk next (Set.insert current visited)

        -- `Just terminal` only when `name` is a passthrough that resolves.
        resolve name =
            if Dict.member name passthrough then
                walk name Set.empty

            else
                Nothing
    in
    groups
        |> List.filter (\g -> resolve g.name == Nothing)
        |> List.map
            (\g ->
                { g
                    | contents =
                        { refs =
                            g.contents.refs
                                |> List.map (\r -> resolve r |> Maybe.withDefault r)
                                |> dedupRefs
                        , items = g.contents.items
                        }
                }
            )


{-| Deduplicate a list of strings, keeping the first occurrence's order. -}
dedupRefs : List String -> List String
dedupRefs refs =
    List.foldl
        (\r ( seen, acc ) ->
            if Set.member r seen then
                ( seen, acc )

            else
                ( Set.insert r seen, r :: acc )
        )
        ( Set.empty, [] )
        refs
        |> Tuple.second
        |> List.reverse


{-| Names of the groups that are fully done: every item they contain — their own
and those of every group they reference, transitively — is in `done`. A group
with no items at all is vacuously done, so a container whose sub-groups are all
finished sinks alongside them. Only meaningful in Mark mode.
-}
doneGroupNames : Set.Set String -> List Group -> Set.Set String
doneGroupNames done groups =
    let
        refAdj =
            groups |> List.map (\g -> ( g.name, g.contents.refs )) |> Dict.fromList

        itemsOf =
            groups |> List.map (\g -> ( g.name, g.contents.items )) |> Dict.fromList

        -- Every item below `name`, its own included. reachable carries a visited
        -- set, so a ref cycle terminates.
        itemsBelow name =
            reachable refAdj name
                |> Set.toList
                |> List.concatMap (\n -> Dict.get n itemsOf |> Maybe.withDefault [])
    in
    groups
        |> List.filter (\g -> itemsBelow g.name |> List.all (\item -> Set.member item done))
        |> List.map .name
        |> Set.fromList


{-| The slice of the list currently on screen, for the heading. -}
viewingLabel : Model -> String
viewingLabel model =
    case ( model.doneDisplay, model.viewMode ) of
        ( Mark, _ ) ->
            "all"

        ( Hide, Done ) ->
            "done"

        ( Hide, _ ) ->
            "to do"


groupsView : Model -> List Group -> List (Html Msg)
groupsView model groups =
    let
        -- Reverse of contents.refs: the groups that directly reference `name`.
        parentsOf name =
            groups
                |> List.filter (\g -> List.member name g.contents.refs)
                |> List.map .name
                |> List.sort

        -- Computed once for the whole render, not per group.
        doneGroups =
            case model.doneDisplay of
                Mark ->
                    doneGroupNames model.done groups

                Hide ->
                    -- Nothing sinks or fades while done work is off the page.
                    Set.empty

        isDoneGroup group =
            Set.member group.name doneGroups
    in
    h3 [] [ text ("Viewing " ++ viewingLabel model) ]
        :: (groups
                -- Done groups sort below the rest, alphabetical within each half.
                |> List.sortBy
                    (\g ->
                        ( if isDoneGroup g then
                            1

                          else
                            0
                        , g.name
                        )
                    )
                |> List.map
                    (\group ->
                        -- id is the group name so nested-group refs can link to it (href="#name")
                        div
                            (id group.name
                                :: (if isDoneGroup group then
                                        -- The block as a whole recedes; its items
                                        -- are not faded again on top of that.
                                        [ style "opacity" "0.5" ]

                                    else
                                        []
                                   )
                            )
                            (let
                                ( todoItems, doneItems ) =
                                    group.contents.items
                                        |> List.partition (\item -> not (Set.member item model.done))

                                -- The items to show, each paired with whether to
                                -- render it faded.
                                items =
                                    case ( model.doneDisplay, model.viewMode ) of
                                        ( Mark, _ ) ->
                                            List.map (\item -> ( item, False )) todoItems
                                                ++ List.map (\item -> ( item, not (isDoneGroup group) )) doneItems

                                        ( Hide, Done ) ->
                                            -- Everything on this page is done, so
                                            -- fading it all would say nothing.
                                            doneItems |> List.map (\item -> ( item, False ))

                                        ( Hide, _ ) ->
                                            todoItems |> List.map (\item -> ( item, False ))

                                toClickableItem ( itemKey, faded ) itemText =
                                    p
                                        ([ style "cursor" "pointer"
                                         , Html.Events.onClick (ItemDone itemKey (not (Set.member itemKey model.done)))
                                         ]
                                            ++ (if faded then
                                                    [ style "opacity" "0.45", style "color" "#555" ]

                                                else
                                                    []
                                               )
                                        )
                                        [ text itemText ]
                             in
                             if model.itemsOnly > 0 then
                                items
                                    |> List.map
                                        (\item ->
                                            toClickableItem item
                                                ((if model.itemsOnly == 1 then
                                                    group.name ++ ":"

                                                  else
                                                    ""
                                                 )
                                                    ++ Tuple.first item
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
                                                                |> List.map (\item -> toClickableItem item (Tuple.first item))
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


type alias SimplifyResponse =
    { selection : String
    , removedRefs : List String
    , removedItems : List String
    }


decodeSimplify : Json.Decode.Decoder SimplifyResponse
decodeSimplify =
    Json.Decode.map3 SimplifyResponse
        (Json.Decode.field "selection" Json.Decode.string)
        (Json.Decode.field "removedRefs" <| decodedWithNullAsDefault [] <| Json.Decode.list Json.Decode.string)
        (Json.Decode.field "removedItems" <| decodedWithNullAsDefault [] <| Json.Decode.list Json.Decode.string)


{-| expectSimplify mirrors expectGroups: decode the JSON simplify response, and
on a non-2xx status surface the server's plain-text error body to the user.
-}
expectSimplify : (Result String SimplifyResponse -> msg) -> Http.Expect msg
expectSimplify toMsg =
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
                    Json.Decode.decodeString decodeSimplify body
                        |> Result.mapError
                            (\err -> "Data received was not in the correct format: " ++ Json.Decode.errorToString err)



--- GRAPH VIEW
--
-- A whole-graph view of the resolved groups and their items. Groups and items
-- are both nodes (coloured differently); each `ref` is a directed edge
-- parent -> child ("parent includes child") and each item is an edge
-- group -> item. Group nodes are laid out in columns by depth (longest path
-- from a root); every item node sits in a single far-right column, aligned.
-- The whole thing is a pan/zoomable SVG, and clicking a node dims everything
-- not adjacent to it.


nodeHeight : Float
nodeHeight =
    22


{-| String.fromFloat, abbreviated because SVG coordinate strings use it a lot. -}
gStr : Float -> String
gStr =
    String.fromFloat


type NodeKind
    = GroupNode
    | ItemNode


type NodeState
    = Selected
    | Neighbour
    | Normal


type alias GNode =
    -- id is unique across kinds ("g:"/"i:" prefixed); label is the display name.
    { id : String, label : String, kind : NodeKind, x : Float, y : Float, w : Float }


type alias GLayout =
    { nodes : List GNode

    -- edges are ( parentId, childId ): a group includes a child group or item.
    , edges : List ( String, String )
    , width : Float
    , height : Float
    }


groupId : String -> String
groupId name =
    "g:" ++ name


itemId : String -> String
itemId name =
    "i:" ++ name


{-| Longest-path layering: a node's layer is one more than the deepest of its
parents (the groups that include it), so roots sit in column 0. Relaxing over
every edge `n` times converges for a DAG and stays bounded if a cycle sneaks in.
-}
computeLayers : List String -> List ( String, String ) -> Dict String Int
computeLayers names edges =
    let
        relaxOnce dict =
            List.foldl
                (\( parent, child ) d ->
                    let
                        parentLayer =
                            Dict.get parent d |> Maybe.withDefault 0

                        childLayer =
                            Dict.get child d |> Maybe.withDefault 0
                    in
                    if parentLayer + 1 > childLayer then
                        Dict.insert child (parentLayer + 1) d

                    else
                        d
                )
                dict
                edges

        iterate n dict =
            if n <= 0 then
                dict

            else
                iterate (n - 1) (relaxOnce dict)
    in
    iterate (List.length names) (names |> List.map (\n -> ( n, 0 )) |> Dict.fromList)


columnPitch : Float
columnPitch =
    240


rowPitch : Float
rowPitch =
    30


{-| Width of a node box sized to its label (roughly monospace character width). -}
nodeW : String -> Float
nodeW label =
    max 44 (toFloat (String.length label) * 6.6 + 20)


{-| Lay a column of node ids out vertically, centred on y = 0. -}
columnNodes : Float -> NodeKind -> (String -> String) -> List String -> List GNode
columnNodes x kind label ids =
    let
        startY =
            -(toFloat (List.length ids - 1) / 2) * rowPitch
    in
    ids
        |> List.indexedMap
            (\i id ->
                { id = id, label = label id, kind = kind, x = x, y = startY + toFloat i * rowPitch, w = nodeW (label id) }
            )


computeLayout : List Group -> GLayout
computeLayout groups =
    let
        pad =
            18

        groupNameSet =
            Set.fromList (List.map .name groups)

        groupIds =
            List.map (.name >> groupId) groups

        -- Group -> group edges, from refs the server actually returned.
        groupEdges =
            groups
                |> List.concatMap
                    (\g ->
                        g.contents.refs
                            |> List.filter (\r -> Set.member r groupNameSet)
                            |> List.map (\r -> ( groupId g.name, groupId r ))
                    )

        groupLayers =
            computeLayers groupIds groupEdges

        maxGroupLayer =
            groupLayers |> Dict.values |> List.maximum |> Maybe.withDefault 0

        -- Group ids bucketed by layer, alphabetical within each column.
        groupBuckets =
            Dict.foldl
                (\id layer acc -> Dict.update layer (\mb -> Just (id :: Maybe.withDefault [] mb)) acc)
                Dict.empty
                groupLayers
                |> Dict.map (\_ ids -> List.sort ids)

        -- Strip the "g:"/"i:" id prefix back to the display name.
        labelOf id =
            String.dropLeft 2 id

        groupRaw =
            groupBuckets
                |> Dict.toList
                |> List.concatMap
                    (\( layer, ids ) -> columnNodes (toFloat layer * columnPitch) GroupNode labelOf ids)

        groupY =
            groupRaw |> List.map (\n -> ( n.id, n.y )) |> Dict.fromList

        -- Group -> item edges (deduped item nodes: one per unique item name).
        itemEdges =
            groups
                |> List.concatMap (\g -> g.contents.items |> List.map (\it -> ( groupId g.name, itemId it )))

        -- Order items in the far-right column by the mean vertical position of
        -- their parent groups, then name, so items sit near their group(s).
        orderedItemIds =
            groups
                |> List.concatMap (\g -> g.contents.items)
                |> Set.fromList
                |> Set.toList
                |> List.sortBy
                    (\name ->
                        let
                            id =
                                itemId name

                            parentYs =
                                itemEdges |> List.filterMap (\( p, c ) -> ifThen (c == id) (Dict.get p groupY))
                        in
                        ( meanOrZero parentYs, name )
                    )
                |> List.map itemId

        itemRaw =
            columnNodes (toFloat (maxGroupLayer + 1) * columnPitch) ItemNode labelOf orderedItemIds

        rawNodes =
            groupRaw ++ itemRaw

        minX =
            rawNodes |> List.map .x |> List.minimum |> Maybe.withDefault 0

        minY =
            rawNodes |> List.map .y |> List.minimum |> Maybe.withDefault 0

        -- Shift everything into the positive quadrant with a margin.
        shifted =
            rawNodes |> List.map (\n -> { n | x = n.x - minX + pad, y = n.y - minY + pad })

        maxX =
            shifted |> List.map (\n -> n.x + n.w) |> List.maximum |> Maybe.withDefault 0

        maxY =
            shifted |> List.map (\n -> n.y + nodeHeight) |> List.maximum |> Maybe.withDefault 0
    in
    { nodes = shifted, edges = groupEdges ++ itemEdges, width = maxX + pad, height = maxY + pad }


ifThen : Bool -> Maybe a -> Maybe a
ifThen cond maybe =
    if cond then
        maybe

    else
        Nothing


{-| Prepend `val` to the list stored at `key` (creating it if absent). -}
pushAdj : String -> String -> Dict String (List String) -> Dict String (List String)
pushAdj key val dict =
    Dict.update key (\mb -> Just (val :: Maybe.withDefault [] mb)) dict


{-| Every node reachable from `start` along `adj` (inclusive of `start`),
breadth/depth-first with a visited set so cycles terminate. -}
reachable : Dict String (List String) -> String -> Set.Set String
reachable adj start =
    reachableHelp adj [ start ] Set.empty


reachableHelp : Dict String (List String) -> List String -> Set.Set String -> Set.Set String
reachableHelp adj frontier visited =
    case frontier of
        [] ->
            visited

        x :: rest ->
            if Set.member x visited then
                reachableHelp adj rest visited

            else
                reachableHelp adj (Maybe.withDefault [] (Dict.get x adj) ++ rest) (Set.insert x visited)


meanOrZero : List Float -> Float
meanOrZero xs =
    case xs of
        [] ->
            0

        _ ->
            List.sum xs / toFloat (List.length xs)


{-| A small coloured square used in the graph legend. -}
legendSwatch : String -> String -> Html Msg
legendSwatch fill stroke =
    span
        [ style "display" "inline-block"
        , style "width" "11px"
        , style "height" "11px"
        , style "background" fill
        , style "border" ("1px solid " ++ stroke)
        , style "border-radius" "2px"
        , style "margin-right" "3px"
        , style "vertical-align" "middle"
        ]
        []


{-| Scale/pan that frames the whole graph within a nominal viewport. The real
SVG pixel size is not known to Elm, so this uses a fixed baseline that lands the
graph sensibly on typical screens; the user can pan/zoom from there.
-}
fitToView : Maybe (List Group) -> { scale : Float, panX : Float, panY : Float }
fitToView maybeGroups =
    case maybeGroups of
        Nothing ->
            { scale = 1, panX = 20, panY = 20 }

        Just groups ->
            let
                layout =
                    computeLayout groups

                baseW =
                    900

                baseH =
                    560

                gW =
                    max 1 layout.width

                gH =
                    max 1 layout.height

                scale =
                    clamp 0.15 2 (min (baseW / gW) (baseH / gH))
            in
            { scale = scale
            , panX = (baseW - gW * scale) / 2
            , panY = (baseH - gH * scale) / 2
            }


{-| Restrict `groups` to the lineage — ancestors ∪ descendants, inclusive — of
the node id `selectedId`: keep only the groups in the lineage and, on those, only
the refs and items that are themselves in the lineage. Re-running computeLayout on
the result yields just the focused subgraph. Always apply this to the full
effectiveResults list, never to an already-filtered subset.
-}
lineageGroups : String -> List Group -> List Group
lineageGroups selectedId groups =
    let
        edges =
            (computeLayout groups).edges

        childrenAdj =
            List.foldl (\( p, c ) d -> pushAdj p c d) Dict.empty edges

        parentsAdj =
            List.foldl (\( p, c ) d -> pushAdj c p d) Dict.empty edges

        focused =
            Set.union (reachable parentsAdj selectedId) (reachable childrenAdj selectedId)
    in
    groups
        |> List.filter (\g -> Set.member (groupId g.name) focused)
        |> List.map
            (\g ->
                { g
                    | contents =
                        { refs = g.contents.refs |> List.filter (\r -> Set.member (groupId r) focused)
                        , items = g.contents.items |> List.filter (\it -> Set.member (itemId it) focused)
                        }
                }
            )


{-| The group list to feed the graph: the whole thing, or — when a node is
focused — just that node's lineage. -}
graphGroups : Model -> List Group -> List Group
graphGroups model groups =
    case model.graphSelected of
        Nothing ->
            groups

        Just sel ->
            lineageGroups sel groups


{-| When a graph node is focused, a pill above the graph names the focused node
and offers a reset; nothing is shown otherwise. The node id carries a "g:"/"i:"
kind prefix, dropped here to show the plain name. -}
graphFocusBanner : Maybe String -> Html Msg
graphFocusBanner selected =
    case selected of
        Nothing ->
            text ""

        Just nodeId ->
            div
                [ style "display" "inline-flex"
                , style "align-items" "center"
                , style "gap" "0.4rem"
                , style "margin" "0 0 0.5rem 0"
                , style "padding" "0.2rem 0.3rem 0.2rem 0.7rem"
                , style "background" "#ebf8ff"
                , style "border" "1px solid #90cdf4"
                , style "border-radius" "999px"
                , style "font-size" "0.85em"
                ]
                [ text ("Showing lineage of \u{2018}" ++ String.dropLeft 2 nodeId ++ "\u{2019}")
                , button
                    [ onClick ClearGraphFocus
                    , style "cursor" "pointer"
                    , Html.Attributes.title "Show the whole graph"
                    ]
                    [ text "✕" ]
                ]


graphView : Model -> List Group -> Html Msg
graphView model groups =
    let
        layout =
            computeLayout groups

        nodeDict =
            layout.nodes |> List.map (\n -> ( n.id, n )) |> Dict.fromList

        transform =
            "translate("
                ++ gStr model.graphPanX
                ++ ","
                ++ gStr model.graphPanY
                ++ ") scale("
                ++ gStr model.graphScale
                ++ ")"

        -- The list is already filtered to the focused lineage (see graphGroups),
        -- so every node/edge renders active; the selected node is still
        -- highlighted via the graphSelected match in viewNode.
        edgeEls =
            layout.edges
                |> List.filterMap
                    (\edge ->
                        Maybe.map2 (\parent child -> viewEdge True parent child)
                            (Dict.get (Tuple.first edge) nodeDict)
                            (Dict.get (Tuple.second edge) nodeDict)
                    )

        nodeEls =
            layout.nodes
                |> List.map (\n -> viewNode model.graphSelected True False n)
    in
    Svg.svg
        [ style "width" "100%"
        , style "height" "72vh"
        , style "border" "1px solid #ddd"
        , style "background" "#fafafa"
        , style "display" "block"
        , style "user-select" "none"
        , style "touch-action" "none"
        , style "cursor"
            (case model.graphDrag of
                Just _ ->
                    "grabbing"

                Nothing ->
                    "grab"
            )
        , Html.Events.on "mousedown" panStartDecoder
        , Html.Events.preventDefaultOn "wheel" wheelDecoder
        ]
        [ Svg.defs []
            [ Svg.marker
                [ SA.id "arrowhead"
                , SA.markerWidth "7"
                , SA.markerHeight "7"
                , SA.refX "6"
                , SA.refY "3"
                , SA.orient "auto"
                , SA.markerUnits "strokeWidth"
                ]
                [ Svg.path [ SA.d "M0,0 L6,3 L0,6 Z", SA.fill "#5b6b7a" ] [] ]
            ]
        , Svg.g [ SA.transform transform ] (edgeEls ++ nodeEls)
        ]


viewEdge : Bool -> GNode -> GNode -> Svg.Svg Msg
viewEdge active parent child =
    let
        x1 =
            parent.x + parent.w

        y1 =
            parent.y + nodeHeight / 2

        x2 =
            child.x

        y2 =
            child.y + nodeHeight / 2

        midX =
            x1 + (x2 - x1) / 2

        d =
            "M "
                ++ gStr x1
                ++ " "
                ++ gStr y1
                ++ " C "
                ++ gStr midX
                ++ " "
                ++ gStr y1
                ++ ", "
                ++ gStr midX
                ++ " "
                ++ gStr y2
                ++ ", "
                ++ gStr x2
                ++ " "
                ++ gStr y2
    in
    Svg.path
        ([ SA.d d
         , SA.fill "none"
         , SA.stroke
            (if active then
                "#5b6b7a"

             else
                "#c9d2da"
            )
         , SA.strokeWidth
            (if active then
                "1.4"

             else
                "1"
            )
         , SA.opacity
            (if active then
                "0.85"

             else
                "0.3"
            )
         ]
            ++ (if active then
                    [ SA.markerEnd "url(#arrowhead)" ]

                else
                    []
               )
        )
        []


{-| Fill/stroke/text colours per node kind and state. Groups are blue-grey,
items green, so the two are distinguishable at a glance. -}
nodeColors : NodeKind -> NodeState -> { fill : String, stroke : String, text : String }
nodeColors kind state =
    case ( kind, state ) of
        ( GroupNode, Selected ) ->
            { fill = "#2b6cb0", stroke = "#1a4971", text = "#ffffff" }

        ( GroupNode, Neighbour ) ->
            { fill = "#bee3f8", stroke = "#63b3ed", text = "#1a202c" }

        ( GroupNode, Normal ) ->
            { fill = "#e2e8f0", stroke = "#a0aec0", text = "#1a202c" }

        ( ItemNode, Selected ) ->
            { fill = "#2f855a", stroke = "#22543d", text = "#ffffff" }

        ( ItemNode, Neighbour ) ->
            { fill = "#9ae6b4", stroke = "#48bb78", text = "#1a202c" }

        ( ItemNode, Normal ) ->
            { fill = "#c6f6d5", stroke = "#68d391", text = "#1a202c" }


viewNode : Maybe String -> Bool -> Bool -> GNode -> Svg.Svg Msg
viewNode selected active isNeighbour n =
    let
        state =
            if selected == Just n.id then
                Selected

            else if isNeighbour then
                Neighbour

            else
                Normal

        colors =
            nodeColors n.kind state
    in
    Svg.g
        [ Html.Events.stopPropagationOn "mousedown" (Json.Decode.succeed ( GraphNodeClicked n.id, True ))
        , style "cursor" "pointer"
        , SA.opacity
            (if active then
                "1"

             else
                "0.18"
            )
        ]
        [ Svg.rect
            [ SA.x (gStr n.x)
            , SA.y (gStr n.y)
            , SA.width (gStr n.w)
            , SA.height (gStr nodeHeight)
            , SA.rx "4"
            , SA.ry "4"
            , SA.fill colors.fill
            , SA.stroke colors.stroke
            , SA.strokeWidth "1"
            ]
            []
        , Svg.text_
            [ SA.x (gStr (n.x + n.w / 2))
            , SA.y (gStr (n.y + nodeHeight / 2 + 4))
            , SA.textAnchor "middle"
            , SA.fontSize "11"
            , SA.fontFamily "monospace"
            , SA.fill colors.text
            , style "pointer-events" "none"
            ]
            [ Svg.text n.label ]
        ]


{-| A `mousedown` on empty canvas seeds a pan with the pointer's screen position.
Node mousedowns stop propagation, so this only fires for background grabs.
-}
panStartDecoder : Json.Decode.Decoder Msg
panStartDecoder =
    Json.Decode.map2 GraphDragStart
        (Json.Decode.field "clientX" Json.Decode.float)
        (Json.Decode.field "clientY" Json.Decode.float)


{-| Wheel-zoom, returning `True` to preventDefault so the page does not scroll.
offsetX/offsetY are the pointer position in SVG user units (no viewBox is set,
so 1 user unit == 1 CSS pixel).
-}
wheelDecoder : Json.Decode.Decoder ( Msg, Bool )
wheelDecoder =
    Json.Decode.map3 (\deltaY offsetX offsetY -> ( GraphWheel deltaY offsetX offsetY, True ))
        (Json.Decode.field "deltaY" Json.Decode.float)
        (Json.Decode.field "offsetX" Json.Decode.float)
        (Json.Decode.field "offsetY" Json.Decode.float)
