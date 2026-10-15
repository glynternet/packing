port module HostedMain exposing (..)

import Browser
import Dict exposing (Dict)
import HostedSync as Sync
import Html exposing (button, div, p, pre, text)
import Html.Attributes exposing (disabled, style)
import Html.Events exposing (onClick)
import Http
import Json.Decode as D
import Json.Encode as E
import Main
import Packing
import Process
import Selection
import Set
import Task
import Time


port storeRecovery : String -> Cmd msg


port recoveryResult : ({ state : String, error : Maybe String } -> msg) -> Sub msg


port refreshRequested : (() -> msg) -> Sub msg


port exportRecovery : String -> Cmd msg


type alias Flags =
    { trip : String, client : String, now : Float, recovery : Maybe String, storageError : Maybe String }


type alias Pending =
    { write : Sync.Write, sending : Bool, error : Maybe String }


type alias Model =
    { ui : Main.Model
    , tripId : String
    , client : String
    , started : Float
    , now : Float
    , next : Int
    , remote : Maybe Sync.Trip
    , baseDraft : String
    , baseRevision : Int
    , epoch : String
    , pending : Dict String Pending
    , dirty : Bool
    , conflict : Bool
    , error : Maybe String
    , unreadableRecovery : Maybe String
    , storageError : Maybe String
    , loading : Bool
    , fetchSerial : Int
    }


type Msg
    = UI Main.Msg
    | Loaded Int (Result Http.Error Sync.Trip)
    | Written String String (Result Http.Error Sync.Ack)
    | SaveDraft Int
    | Refresh
    | Retry
    | RetryAt Float
    | UseShared
    | KeepDraft
    | Export
    | Stored { state : String, error : Maybe String }


main =
    Browser.document { init = init, update = update, subscriptions = subscriptions, view = view }


init : Flags -> ( Model, Cmd Msg )
init flags =
    let
        ui =
            Main.defaultModel

        ( empty, _ ) =
            Selection.init (Just { selection = "", groups = [] }) "" Nothing

        base =
            { ui = { ui | selection = empty, hosted = True }
            , tripId = flags.trip
            , client = flags.client
            , started = flags.now
            , now = flags.now
            , next = 0
            , remote = Nothing
            , baseDraft = ""
            , baseRevision = 0
            , epoch = ""
            , pending = Dict.empty
            , dirty = False
            , conflict = False
            , error = Nothing
            , unreadableRecovery = Nothing
            , storageError = flags.storageError
            , loading = False
            , fetchSerial = 0
            }

        restored =
            case flags.recovery of
                Nothing ->
                    base

                Just raw ->
                    case D.decodeString (recoveryDecoder base) raw of
                        Ok cached ->
                            { cached | conflict = D.decodeString (D.field "conflict" D.bool) raw |> Result.withDefault False }

                        Err _ ->
                            { base | error = Just "Could not read device recovery data. Export it before clearing browser storage.", conflict = True, unreadableRecovery = Just raw }
    in
    refresh restored


baseURL : Model -> String
baseURL model =
    "/api/v1/sessions/" ++ model.tripId


refresh : Model -> ( Model, Cmd Msg )
refresh model =
    let
        serial =
            model.fetchSerial + 1
    in
    ( { model | loading = True, fetchSerial = serial }
    , Http.get { url = baseURL model, expect = Http.expectJson (Loaded serial) Sync.tripDecoder }
    )


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        Refresh ->
            refresh model

        Loaded serial result ->
            if serial /= model.fetchSerial then
                ( model, Cmd.none )

            else
                case result of
                    Err err ->
                        ( { model | loading = False, error = Just (errorText err) }, Cmd.none )

                    Ok trip ->
                        reconcile trip
                            { model
                                | loading = False
                                , error =
                                    if model.unreadableRecovery == Nothing then
                                        Nothing

                                    else
                                        model.error
                            }

        UI (Main.SelectionMsg selectionMsg) ->
            if model.remote == Nothing || Dict.member "apply" model.pending then
                ( model, Cmd.none )

            else
                case selectionMsg of
                    Selection.Apply ->
                        case model.remote of
                            Just trip ->
                                if model.dirty || model.conflict || not (Dict.isEmpty model.pending) || not (Selection.canApply model.ui.selection) then
                                    ( model, Cmd.none )

                                else
                                    send "apply" (Sync.write (mutationId model) "apply" trip) model

                            Nothing ->
                                ( model, Cmd.none )

                    _ ->
                        let
                            ( changed, command ) =
                                selection selectionMsg model

                            dirty =
                                changed.ui.selection.draftText /= changed.baseDraft

                            updated =
                                { changed | dirty = dirty }
                        in
                        ( updated
                        , Cmd.batch [ command, persist updated, Process.sleep 500 |> Task.perform (\_ -> SaveDraft updated.ui.selection.revision) ]
                        )

        UI (Main.ItemDone name packed) ->
            case model.remote of
                Just trip ->
                    if model.ui.reviewTab /= Main.Current || not (Dict.isEmpty model.pending) || model.conflict then
                        ( model, Cmd.none )

                    else
                        case Dict.values trip.items |> List.filter (\item -> item.name == name) |> List.head of
                            Just item ->
                                let
                                    w =
                                        Sync.write (mutationId model) "item" trip
                                in
                                send ("item-" ++ item.id) { w | item = item.id, packed = packed, etag = Sync.tag trip ("item-" ++ item.id) item.revision } model

                            Nothing ->
                                ( model, Cmd.none )

                Nothing ->
                    ( model, Cmd.none )

        UI Main.ClearDone ->
            ( model, Cmd.none )

        UI other ->
            let
                ( ui, command ) =
                    Main.update other model.ui
            in
            ( { model | ui = ui }, Cmd.map UI command )

        SaveDraft revision ->
            if revision /= model.ui.selection.revision || not model.dirty || model.conflict || Dict.member "draft" model.pending then
                ( model, Cmd.none )

            else
                saveDraft model

        Written key id result ->
            case Dict.get key model.pending of
                Nothing ->
                    ( model, Cmd.none )

                Just pending ->
                    if pending.write.mutationId /= id then
                        ( model, Cmd.none )

                    else
                        case result of
                            Err err ->
                                let
                                    conflicted =
                                        case err of
                                            Http.BadStatus status ->
                                                status == 409 || status == 412

                                            _ ->
                                                False

                                    updated =
                                        { model | pending = Dict.insert key { pending | sending = False, error = Just (errorText err) } model.pending, conflict = model.conflict || conflicted }

                                    ( refreshing, cmd ) =
                                        if conflicted then
                                            refresh updated

                                        else
                                            ( updated, Cmd.none )
                                in
                                ( refreshing, Cmd.batch [ cmd, persist refreshing ] )

                            Ok ack ->
                                if model.remote |> Maybe.map (\trip -> trip.epoch /= ack.trip.epoch) |> Maybe.withDefault False then
                                    ( { model | conflict = True, pending = Dict.insert key { pending | sending = False, error = Just "The dataset changed while saving. Refresh and review this edit." } model.pending }, Cmd.none )

                                else
                                    let
                                        next =
                                            { model | pending = Dict.remove key model.pending }

                                        acknowledged =
                                            if key == "draft" then
                                                { next | baseDraft = pending.write.text, baseRevision = ack.draftRevision, dirty = next.ui.selection.draftText /= pending.write.text }

                                            else
                                                next

                                        ( merged, cmd ) =
                                            reconcile ack.trip acknowledged

                                        ( saving, save ) =
                                            if merged.dirty && not merged.conflict && not (Dict.member "draft" merged.pending) then
                                                saveDraft merged

                                            else
                                                ( merged, Cmd.none )
                                    in
                                    ( saving, Cmd.batch [ cmd, save, persist saving ] )

        Retry ->
            ( model, Time.now |> Task.perform (Time.posixToMillis >> toFloat >> RetryAt) )

        RetryAt now ->
            if model.conflict then
                refresh model

            else if now - model.started >= 6 * 24 * 60 * 60 * 1000 then
                ( { model | conflict = True, error = Just "These edits are too old to retry safely. Refresh and review them against the shared trip." }, Cmd.none )

            else
                let
                    retry pending =
                        { pending | sending = True, error = Nothing }

                    updated =
                        { model | pending = Dict.map (\_ -> retry) model.pending }
                in
                ( updated, Cmd.batch (persist updated :: (Dict.toList updated.pending |> List.map (\( key, pending ) -> request model key pending.write))) )

        UseShared ->
            if List.any .sending (Dict.values model.pending) then
                ( model, Cmd.none )

            else
                case model.remote of
                    Nothing ->
                        refresh model

                    Just trip ->
                        reconcile trip { model | pending = Dict.empty, dirty = False, conflict = False, epoch = trip.epoch, error = Nothing, unreadableRecovery = Nothing }

        KeepDraft ->
            if List.any .sending (Dict.values model.pending) then
                ( model, Cmd.none )

            else
                case model.remote of
                    Nothing ->
                        refresh model

                    Just trip ->
                        -- This is an explicit user decision after viewing the shared
                        -- text. Conditional writes still detect another intervening edit.
                        saveDraft { model | pending = Dict.empty, conflict = False, epoch = trip.epoch, baseDraft = trip.draft, baseRevision = trip.draftRevision, dirty = model.ui.selection.draftText /= trip.draft, error = Nothing }

        Export ->
            ( model, exportRecovery (Maybe.withDefault (encodeRecovery model) model.unreadableRecovery) )

        Stored result ->
            if result.state == encodeRecovery model then
                ( { model | storageError = result.error }, Cmd.none )

            else
                ( model, Cmd.none )


mutationId : Model -> String
mutationId model =
    model.client ++ "-" ++ String.fromInt model.next


saveDraft : Model -> ( Model, Cmd Msg )
saveDraft model =
    case model.remote of
        Nothing ->
            ( model, Cmd.none )

        Just trip ->
            let
                w =
                    Sync.write (mutationId model) "draft" trip
            in
            send "draft" { w | text = model.ui.selection.draftText, epoch = model.epoch, etag = Sync.tag { trip | epoch = model.epoch } "draft" model.baseRevision } model


send : String -> Sync.Write -> Model -> ( Model, Cmd Msg )
send key w model =
    let
        ui =
            model.ui

        done =
            case model.remote |> Maybe.andThen (\trip -> Dict.get w.item trip.items) of
                Just item ->
                    if w.kind == "item" then
                        if w.packed then
                            Set.insert item.name ui.done

                        else
                            Set.remove item.name ui.done

                    else
                        ui.done

                Nothing ->
                    ui.done

        updated =
            { model | ui = { ui | done = done }, pending = Dict.insert key { write = w, sending = True, error = Nothing } model.pending, next = model.next + 1 }
    in
    ( updated, Cmd.batch [ persist updated, request updated key w ] )


request : Model -> String -> Sync.Write -> Cmd Msg
request model key w =
    Http.request
        { method =
            if w.kind == "apply" then
                "POST"

            else
                "PUT"
        , headers = [ Http.header "If-Match" w.etag, Http.header "X-Packing-Request" "1" ]
        , url =
            baseURL model
                ++ (if w.kind == "item" then
                        "/items/" ++ w.item

                    else
                        "/" ++ w.kind
                   )
        , body = Http.jsonBody (Sync.encodeWrite w)
        , expect = Http.expectJson (Written key w.mutationId) Sync.ackDecoder
        , timeout = Just 15000
        , tracker = Nothing
        }


{-| Fetches cannot replace local intent or move the accepted server revision
backwards. A restore changes epoch and demands explicit recovery.
-}
reconcile : Sync.Trip -> Model -> ( Model, Cmd Msg )
reconcile incoming model =
    if incoming.id /= model.tripId then
        ( model, Cmd.none )

    else
        let
            trip =
                Sync.newest incoming model.remote

            changedEpoch =
                model.epoch /= "" && model.epoch /= trip.epoch

            hasIntent =
                model.dirty || not (Dict.isEmpty model.pending)

            preserve =
                hasIntent || model.conflict

            draft =
                if preserve then
                    model.ui.selection.draftText

                else
                    trip.draft

            ui =
                model.ui

            current =
                ui.selection

            -- Keep request tokens monotonic across fetches: an old preview must
            -- never match a newly initialized model's reused token.
            baseSelection =
                { current | committed = Just trip.snapshot }

            ( selectionModel, effects ) =
                Selection.update (Selection.Edit draft) baseSelection

            done =
                trip.items |> Dict.values |> List.filter .packed |> List.map .name |> Set.fromList

            optimistic =
                Dict.values model.pending
                    |> List.foldl
                        (\pending acc ->
                            case Dict.get pending.write.item trip.items of
                                Just item ->
                                    if pending.write.kind == "item" then
                                        if pending.write.packed then
                                            Set.insert item.name acc

                                        else
                                            Set.remove item.name acc

                                    else
                                        acc

                                Nothing ->
                                    acc
                        )
                        done

            updated =
                { model
                    | remote = Just trip
                    , epoch =
                        if preserve && changedEpoch then
                            model.epoch

                        else
                            trip.epoch
                    , baseDraft =
                        if preserve then
                            model.baseDraft

                        else
                            trip.draft
                    , baseRevision =
                        if preserve then
                            model.baseRevision

                        else
                            trip.draftRevision
                    , conflict = model.conflict || (changedEpoch && hasIntent)
                    , ui =
                        { ui
                            | selection = selectionModel
                            , done = optimistic
                            , selectedGroup = Nothing
                            , reviewTab =
                                if draft == trip.snapshot.selection then
                                    Main.Current

                                else
                                    Main.Changes
                        }
                }
        in
        ( updated, Cmd.batch [ selectionEffects updated effects, persist updated ] )


selection : Selection.Msg -> Model -> ( Model, Cmd Msg )
selection msg model =
    let
        ( _, effects ) =
            Selection.update msg model.ui.selection

        ( ui, _ ) =
            Main.updateSelection msg model.ui

        updated =
            { model | ui = ui }
    in
    ( updated, selectionEffects updated effects )


selectionEffects : Model -> List Selection.Effect -> Cmd Msg
selectionEffects model effects =
    let
        run effect =
            case effect of
                Selection.Debounce revision ->
                    Process.sleep 250 |> Task.perform (\_ -> UI (Main.SelectionMsg (Selection.DebounceElapsed revision)))

                Selection.Fetch token content ->
                    post model "/preview" content (Http.expectJson (Result.mapError errorText >> Selection.Resolved token >> Main.SelectionMsg >> UI) Packing.decodeGroups)

                Selection.SimplifyFetch token content ->
                    post model "/simplify" content (Http.expectJson (Result.mapError errorText >> Selection.Simplified token >> Main.SelectionMsg >> UI) Main.decodeSimplify)

                Selection.Persist ->
                    Cmd.none

                Selection.BootstrapFetch _ _ ->
                    Cmd.none
    in
    Cmd.batch (List.map run effects)


post : Model -> String -> String -> Http.Expect Msg -> Cmd Msg
post model path content expect =
    Http.request { method = "POST", headers = [ Http.header "X-Packing-Request" "1" ], url = baseURL model ++ path, body = Http.jsonBody (E.object [ ( "text", E.string content ) ]), expect = expect, timeout = Just 10000, tracker = Nothing }


errorText : Http.Error -> String
errorText error =
    case error of
        Http.BadStatus 401 ->
            "Access expired. Open the private access link again; your pending edits remain on this device."

        Http.BadStatus 403 ->
            "Access was refused. Your edits have not been acknowledged."

        Http.BadStatus 409 ->
            "The trip changed. Review the shared state before editing again."

        Http.BadStatus 412 ->
            "Changed on another device — review before replacing shared work."

        Http.BadStatus 422 ->
            "This selection cannot be resolved. The draft can still be saved."

        Http.BadStatus 413 ->
            "The selection is too large to save."

        _ ->
            "Save or refresh failed. Edits remain unsynced on this device. Retry when connected."


persist : Model -> Cmd Msg
persist model =
    if model.unreadableRecovery /= Nothing then
        Cmd.none

    else
        storeRecovery (encodeRecovery model)


encodeRecovery : Model -> String
encodeRecovery model =
    E.encode 0
        (E.object
            [ ( "version", E.int 1 )
            , ( "trip", E.string model.tripId )
            , ( "client", E.string model.client )
            , ( "started", E.float model.started )
            , ( "next", E.int model.next )
            , ( "epoch", E.string model.epoch )
            , ( "draft", E.string model.ui.selection.draftText )
            , ( "baseDraft", E.string model.baseDraft )
            , ( "baseRevision", E.int model.baseRevision )
            , ( "dirty", E.bool model.dirty )
            , ( "conflict", E.bool model.conflict )
            , ( "writes", E.list Sync.encodeWrite (Dict.values model.pending |> List.map .write) )
            ]
        )


recoveryDecoder : Model -> D.Decoder Model
recoveryDecoder base =
    D.map2 Tuple.pair (D.field "version" D.int) (D.field "trip" D.string)
        |> D.andThen
            (\( version, trip ) ->
                if version /= 1 || trip /= base.tripId then
                    D.fail "Wrong recovery schema or trip"

                else
                    D.map8
                        (\_ started _ epoch draft baseDraft baseRevision writes ->
                            let
                                ui =
                                    base.ui

                                selectionModel =
                                    ui.selection

                                key w =
                                    if w.kind == "item" then
                                        "item-" ++ w.item

                                    else
                                        w.kind
                            in
                            { base
                                | client = base.client
                                , started =
                                    if List.isEmpty writes then
                                        base.started

                                    else
                                        started
                                , next = base.next
                                , epoch = epoch
                                , baseDraft = baseDraft
                                , baseRevision = baseRevision
                                , ui = { ui | selection = { selectionModel | draftText = draft } }
                                , dirty = draft /= baseDraft
                                , pending = writes |> List.map (\w -> ( key w, { write = w, sending = False, error = Just "Recovered unsynced edit. Retry to check its acknowledgement." } )) |> Dict.fromList
                            }
                        )
                        (D.field "client" D.string)
                        (D.field "started" D.float)
                        (D.field "next" D.int)
                        (D.field "epoch" D.string)
                        (D.field "draft" D.string)
                        (D.field "baseDraft" D.string)
                        (D.field "baseRevision" D.int)
                        (D.field "writes" (D.list Sync.writeDecoder))
            )


subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.batch [ Main.graphSubscriptions model.ui |> Sub.map UI, refreshRequested (\_ -> Refresh), recoveryResult Stored ]


view : Model -> Browser.Document Msg
view model =
    let
        ui =
            model.ui

        busy =
            not (Dict.isEmpty model.pending)

        doc =
            Main.view
                { ui
                    | serverState =
                        if model.remote == Nothing then
                            Main.LoadingSelection

                        else if Dict.member "apply" model.pending then
                            Main.SavingSelection

                        else
                            Main.LocalOnly
                    , hostedApplyLocked = busy || model.dirty || model.conflict || model.loading
                    , hostedPackingLocked = busy || model.conflict || model.remote == Nothing
                }

        errors =
            model.error :: model.storageError :: (Dict.values model.pending |> List.map .error)

        status =
            if model.remote == Nothing then
                "Trip not loaded"

            else if model.loading then
                "Refreshing…"

            else if model.conflict then
                "Conflicting edits — review required"

            else if List.any .sending (Dict.values model.pending) then
                "Saving…"

            else if busy || model.dirty then
                "Unsynced on this device"

            else if model.storageError /= Nothing then
                "Saved on server; device recovery unavailable"

            else
                "Saved"

        sending =
            List.any .sending (Dict.values model.pending)
    in
    { title = "Private packing trip"
    , body =
        div [ style "padding" "1rem", Html.Attributes.attribute "aria-label" "Trip sync" ]
            ([ p [ Html.Attributes.attribute "role" "status" ] [ text status ]
             , button [ onClick Refresh ] [ text "Refresh trip" ]
             , button [ onClick Retry, disabled (sending || not busy || model.conflict) ] [ text "Retry unsaved edits" ]
             , button [ onClick Export ] [ text "Export device recovery" ]
             ]
                ++ (errors |> List.filterMap identity |> List.map (\err -> p [ style "color" "#a32121" ] [ text err ]))
                ++ (if model.conflict || (busy && not sending) then
                        [ p [] [ text "Your draft remains in the editor. Shared draft:" ]
                        , pre [ style "white-space" "pre-wrap" ] [ text (model.remote |> Maybe.map .draft |> Maybe.withDefault "") ]
                        , button [ onClick UseShared, disabled sending ] [ text "Discard device edits and use shared trip" ]
                        , button [ onClick KeepDraft, disabled (sending || List.any (\p -> p.write.kind == "item") (Dict.values model.pending) || model.unreadableRecovery /= Nothing) ] [ text "Save my draft over this shared draft" ]
                        ]

                    else
                        []
                   )
            )
            :: List.map (Html.map UI) doc.body
    }
