module SelectionTest exposing (tests)

import Expect
import Json.Decode as Decode
import Main
import Packing exposing (Group, Snapshot)
import Selection exposing (Effect(..), Msg(..), Preview(..))
import Set
import StoredState
import Test exposing (Test, describe, test)


group : String -> List String -> List String -> Group
group name refs items =
    { name = name, contents = { refs = refs, items = items } }


baseline : Snapshot
baseline =
    { selection = "ref: sun\nTowel"
    , groups = [ group "sun" [] [ "Sunscreen", "Sunglasses" ], group "Individual Items" [] [ "Towel" ] ]
    }


step : Msg -> Selection.Model -> Selection.Model
step msg =
    Selection.update msg >> Tuple.first


settle : Result String (List Group) -> Selection.Model -> Selection.Model
settle result model =
    case model.preview of
        Loading token ->
            step (Resolved token result) model

        _ ->
            model


initial : Selection.Model
initial =
    Selection.init (Just baseline) baseline.selection Nothing
        |> Tuple.first
        |> settle (Ok baseline.groups)


draft : String -> List Group -> Selection.Model
draft selection groups =
    initial |> step (Edit selection) |> step Resolve |> settle (Ok groups)


tests : Test
tests =
    describe "Selection drafts"
        [ test "successful previews leave the accepted snapshot intact until Apply" <|
            \_ ->
                let
                    preview =
                        draft "ref: cold" [ group "cold" [] [ "Gloves" ] ]

                    ( applied, effects ) =
                        Selection.update Apply preview
                in
                Expect.equal
                    ( Just baseline, Selection.ready preview, [ Persist ] )
                    ( preview.committed, applied.committed, effects )
        , test "an invalid or pending draft cannot be applied" <|
            \_ ->
                let
                    pending =
                        initial |> step (Edit "ref:") |> step Resolve

                    failed =
                        pending |> settle (Err "empty reference value")
                in
                Expect.equal
                    ( pending, failed )
                    ( step Apply pending, step Apply failed )
        , test "Discard restores the exact accepted snapshot with no fetch" <|
            \_ ->
                let
                    changed =
                        draft "ref: cold" [ group "cold" [] [ "Gloves" ] ]

                    ( discarded, effects ) =
                        Selection.update Discard changed
                in
                Expect.equal
                    ( baseline.selection, Just baseline, [ Persist ] )
                    ( discarded.draftText, Selection.ready discarded, effects )
        , test "Discard invalidates in-flight resolution and debounce callbacks" <|
            \_ ->
                let
                    pending =
                        initial |> step (Edit "ref: cold") |> step Resolve

                    discarded =
                        step Discard pending
                in
                case pending.preview of
                    Loading token ->
                        Expect.equal discarded
                            (discarded
                                |> step (Resolved token (Ok [ group "cold" [] [ "Gloves" ] ]))
                                |> step (DebounceElapsed pending.revision)
                            )

                    _ ->
                        Expect.fail "Expected an in-flight request"
        , test "only the latest request can update the same draft revision" <|
            \_ ->
                let
                    first =
                        initial |> step Resolve

                    second =
                        first |> step Resolve
                in
                case first.preview of
                    Loading token ->
                        Expect.equal second (step (Resolved token (Err "old error")) second)

                    _ ->
                        Expect.fail "Expected an in-flight request"
        , test "editing away and back to the same text invalidates old responses" <|
            \_ ->
                let
                    pending =
                        initial |> step Resolve

                    changed =
                        pending |> step (Edit "Passport") |> step (Edit baseline.selection) |> step Resolve
                in
                case pending.preview of
                    Loading token ->
                        Expect.equal changed (step (Resolved token (Ok [])) changed)

                    _ ->
                        Expect.fail "Expected an in-flight request"
        , test "Simplify cannot overwrite a subsequent edit or Discard" <|
            \_ ->
                let
                    pending =
                        step Simplify initial

                    edited =
                        step (Edit "Passport") pending

                    discarded =
                        step Discard pending
                in
                case pending.pendingSimplify of
                    Just token ->
                        let
                            late =
                                Simplified token (Ok { selection = "", removedRefs = [ "sun" ], removedItems = [ "Towel" ] })
                        in
                        Expect.equal ( edited, discarded ) ( step late edited, step late discarded )

                    Nothing ->
                        Expect.fail "Expected a Simplify request"
        , test "Apply is blocked while Simplify is pending" <|
            \_ ->
                let
                    pending =
                        draft "Towel" [ group "Individual Items" [] [ "Towel" ] ] |> step Simplify
                in
                Expect.equal pending (step Apply pending)
        , test "a failed Simplify leaves a valid preview usable" <|
            \_ ->
                let
                    pending =
                        draft "Towel" [ group "Individual Items" [] [ "Towel" ] ] |> step Simplify
                in
                case pending.pendingSimplify of
                    Just token ->
                        let
                            failed =
                                step (Simplified token (Err "offline")) pending
                        in
                        Expect.equal ( True, Selection.ready pending, Just "offline" )
                            ( Selection.canApply failed, Selection.ready failed, failed.simplifyError )

                    Nothing ->
                        Expect.fail "Expected a Simplify request"
        , test "Simplify creates a reviewable draft and preserves accepted text" <|
            \_ ->
                let
                    pending =
                        step Simplify initial
                in
                case pending.pendingSimplify of
                    Just token ->
                        let
                            simplified =
                                step (Simplified token (Ok { selection = "ref: sun", removedRefs = [], removedItems = [ "Towel" ] })) pending
                        in
                        Expect.equal ( "ref: sun", Just baseline, Nothing )
                            ( simplified.draftText, simplified.committed, Selection.ready simplified )

                    Nothing ->
                        Expect.fail "Expected a Simplify request"
        , test "comment-only changes can be applied despite having the same items" <|
            \_ ->
                Expect.equal True (Selection.canApply (draft (baseline.selection ++ "\n# comment") baseline.groups))
        , test "empty selections are valid and can be accepted" <|
            \_ ->
                Expect.equal (Just { selection = "", groups = [] }) (draft "" [] |> step Apply |> .committed)
        , test "server ordering and duplicate membership do not create dirty state" <|
            \_ ->
                let
                    groups =
                        [ group "Individual Items" [] [ "Towel", "Towel" ]
                        , group "sun" [] [ "Sunglasses", "Sunscreen" ]
                        ]
                in
                Expect.equal False (draft baseline.selection groups |> Selection.hasChanges)
        , test "group-file changes with unchanged text require acceptance" <|
            \_ ->
                let
                    changed =
                        draft baseline.selection [ group "sun" [] [ "Hat" ] ]
                in
                Expect.equal ( True, Just baseline ) ( Selection.canApply changed, changed.committed )
        , test "changed reference structure can be accepted without an item delta" <|
            \_ ->
                Expect.equal True
                    (draft baseline.selection (group "container" [ "sun" ] [] :: baseline.groups) |> Selection.canApply)
        , test "a late migration resolves the old baseline without replacing a newer draft" <|
            \_ ->
                let
                    migrating =
                        Selection.init Nothing baseline.selection (Just baseline.selection) |> Tuple.first

                    edited =
                        migrating |> step (Edit "Passport") |> step Resolve |> settle (Ok [ group "Individual Items" [] [ "Passport" ] ])
                in
                case migrating.bootstrap of
                    Just bootstrap ->
                        let
                            migrated =
                                step (BootstrapResolved bootstrap.token (Ok baseline.groups)) edited
                        in
                        Expect.equal
                            { committed = Just baseline, text = "Passport", ready = Selection.ready edited, canApply = True }
                            { committed = migrated.committed, text = migrated.draftText, ready = Selection.ready migrated, canApply = Selection.canApply migrated }

                    Nothing ->
                        Expect.fail "Expected migration request"
        , test "failed migration keeps saved text recoverable without an accepted baseline" <|
            \_ ->
                let
                    migrating =
                        Selection.init Nothing "ref: missing" (Just "ref: missing") |> Tuple.first
                in
                case migrating.bootstrap of
                    Just bootstrap ->
                        let
                            failed =
                                step (BootstrapResolved bootstrap.token (Err "missing group")) migrating
                        in
                        Expect.equal ( "ref: missing", Nothing, False )
                            ( failed.draftText, failed.committed, Selection.canApply failed )

                    Nothing ->
                        Expect.fail "Expected migration request"
        , storageTests
        , mainTests
        ]


storageTests : Test
storageTests =
    describe "Storage"
        [ test "v2 reload preserves accepted groups, an invalid draft, and packing progress" <|
            \_ ->
                let
                    saved =
                        initial |> step (Edit "ref:")
                in
                StoredState.encode saved [ "Sunscreen" ] "hidden"
                    |> Decode.decodeString (StoredState.decoder "default")
                    |> Expect.equal
                        (Ok
                            { committed = Just baseline
                            , draftText = "ref:"
                            , legacySelection = Nothing
                            , done = [ "Sunscreen" ]
                            , doneDisplay = Just "hidden"
                            }
                        )
        , test "legacy state is decoded for baseline migration" <|
            \_ ->
                Decode.decodeString (StoredState.decoder "default") "{\"selection\":\"Towel\",\"done\":[\"Towel\"]}"
                    |> Result.map (\stored -> ( stored.legacySelection, stored.draftText, stored.done ))
                    |> Expect.equal (Ok ( Just "Towel", "Towel", [ "Towel" ] ))
        , test "an interrupted migration retains both its old selection and the new draft" <|
            \_ ->
                let
                    migrating =
                        Selection.init Nothing "Towel" (Just "Towel") |> Tuple.first |> step (Edit "Passport")
                in
                StoredState.encode migrating [] "marked"
                    |> Decode.decodeString (StoredState.decoder "default")
                    |> Result.map (\stored -> ( stored.legacySelection, stored.draftText ))
                    |> Expect.equal (Ok ( Just "Towel", "Passport" ))
        , test "malformed or future state fails explicitly" <|
            \_ ->
                [ "{\"version\":3}", "{\"version\":2,\"committed\":{},\"draftText\":\"Towel\"}", "{\"selection\":42}" ]
                    |> List.map (Decode.decodeString (StoredState.decoder "default") >> Result.toMaybe)
                    |> Expect.equal [ Nothing, Nothing, Nothing ]
        ]


mainTests : Test
mainTests =
    describe "Packing progress integration"
        [ test "Apply and Discard preserve packing progress, including items absent from the draft" <|
            \_ ->
                let
                    model =
                        Main.defaultModel

                    edited =
                        { model | selection = draft "Passport" [ group "Individual Items" [] [ "Passport" ] ], done = Set.fromList [ "Towel" ] }

                    applied =
                        Main.update (Main.SelectionMsg Apply) edited |> Tuple.first

                    discarded =
                        Main.update (Main.SelectionMsg Discard) edited |> Tuple.first
                in
                Expect.equal ( edited.done, edited.done ) ( applied.done, discarded.done )
        , test "proposed lists cannot change packing progress" <|
            \_ ->
                let
                    base =
                        Main.defaultModel

                    model =
                        { base | reviewTab = Main.Proposed, done = Set.singleton "Towel" }

                    changed =
                        model
                            |> Main.update (Main.ItemDone "Sunscreen" True)
                            |> Tuple.first
                            |> Main.update Main.ClearDone
                            |> Tuple.first
                in
                Expect.equal model.done changed.done
        , test "stale storage acknowledgements cannot hide a current save failure" <|
            \_ ->
                let
                    base =
                        Main.defaultModel

                    model =
                        { base | storageError = Just "quota exceeded" }
                in
                Main.update (Main.StorageResult { state = "old state", error = Nothing }) model
                    |> Tuple.first
                    |> .storageError
                    |> Expect.equal model.storageError
        ]
