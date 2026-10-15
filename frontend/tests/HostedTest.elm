module HostedTest exposing (tests)

import Dict
import Expect
import HostedMain as Hosted
import HostedSync as Sync
import Http
import Json.Decode as D
import Main
import Selection
import Test exposing (describe, test)


trip : Sync.Trip
trip =
    { id = "trip"
    , epoch = "epoch"
    , revision = 1
    , draft = "Tent"
    , draftRevision = 1
    , generation = 1
    , snapshot = { selection = "Tent", groups = [] }
    , items = Dict.empty
    }


initial : Hosted.Model
initial =
    Hosted.init { trip = "trip", client = "client1234567890123", now = 1000, recovery = Nothing, storageError = Nothing }
        |> Tuple.first
        |> step (Hosted.Loaded 1 (Ok trip))


step : Hosted.Msg -> Hosted.Model -> Hosted.Model
step msg =
    Hosted.update msg >> Tuple.first


edit : String -> Hosted.Model -> Hosted.Model
edit content =
    step (Hosted.UI (Main.SelectionMsg (Selection.Edit content)))


save : Hosted.Model -> Hosted.Model
save model =
    step (Hosted.SaveDraft model.ui.selection.revision) model


tests =
    describe "Hosted HTTP reconciliation"
        [ test "refresh retains unsaved text and its original conditional version" <|
            \_ ->
                let
                    local =
                        initial |> edit "Local draft"

                    remote =
                        { trip | revision = 5, draft = "Other device", draftRevision = 3 }

                    result =
                        local |> step (Hosted.Loaded 1 (Ok remote))
                in
                Expect.equal ( "Local draft", 1, ( Just 5, True ) ) ( result.ui.selection.draftText, result.baseRevision, ( Maybe.map .revision result.remote, result.dirty ) )
        , test "typing during save retains newer text and sends it under the acknowledged version" <|
            \_ ->
                let
                    saving =
                        initial |> edit "First draft" |> save

                    id =
                        Dict.get "draft" saving.pending |> Maybe.map (.write >> .mutationId) |> Maybe.withDefault ""

                    remote =
                        { trip | revision = 2, draft = "First draft", draftRevision = 2 }

                    result =
                        saving |> edit "Newer text" |> step (Hosted.Written "draft" id (Ok { draftRevision = 2, generation = 1, revision = 2, trip = remote }))

                    sent =
                        Dict.get "draft" result.pending |> Maybe.map .write
                in
                Expect.equal ( "Newer text", Just "Newer text", Just (Sync.tag trip "draft" 2) ) ( result.ui.selection.draftText, Maybe.map .text sent, Maybe.map .etag sent )
        , test "late aggregate responses cannot regress confirmed state" <|
            \_ ->
                let
                    remote =
                        { trip | revision = 9, draftRevision = 4, draft = "Latest" }
                in
                Expect.equal 9 (Sync.newest trip (Just remote)).revision
        , test "recovery retains mutation identity and does not mark it acknowledged" <|
            \_ ->
                let
                    saved =
                        initial |> edit "Unsent or unacknowledged" |> save

                    decoded =
                        D.decodeString (Hosted.recoveryDecoder initial) (Hosted.encodeRecovery saved)
                in
                case decoded of
                    Err error ->
                        Expect.fail (D.errorToString error)

                    Ok recovered ->
                        Expect.equal
                            ( Dict.values saved.pending |> List.map (.write >> .mutationId), True, [ False ] )
                            ( Dict.values recovered.pending |> List.map (.write >> .mutationId), recovered.dirty, Dict.values recovered.pending |> List.map .sending )
        , test "restored dataset requires review without dropping local draft" <|
            \_ ->
                let
                    result =
                        initial |> edit "Local" |> step (Hosted.Loaded 1 (Ok { trip | epoch = "restored" }))
                in
                Expect.equal ( True, "Local", "epoch" ) ( result.conflict, result.ui.selection.draftText, result.epoch )
        , test "stale fetch token is ignored" <|
            \_ ->
                let
                    result =
                        initial |> step Hosted.Refresh |> step (Hosted.Loaded 1 (Ok { trip | draft = "Stale" }))
                in
                Expect.equal "Tent" result.ui.selection.draftText
        , test "failed acknowledgement preserves edit and retry ID" <|
            \_ ->
                let
                    saving =
                        initial |> edit "Local" |> save

                    id =
                        Dict.get "draft" saving.pending |> Maybe.map (.write >> .mutationId) |> Maybe.withDefault ""

                    result =
                        saving |> step (Hosted.Written "draft" id (Err Http.NetworkError))
                in
                Expect.equal ( "Local", Just id, Just False )
                    ( result.ui.selection.draftText, Dict.get "draft" result.pending |> Maybe.map (.write >> .mutationId), Dict.get "draft" result.pending |> Maybe.map .sending )
        ]
