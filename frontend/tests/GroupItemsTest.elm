module GroupItemsTest exposing (tests)

import Expect
import Main
import Packing exposing (Snapshot)
import Selection
import Set
import Test exposing (Test, describe, test)
import Test.Html.Query as Query
import Test.Html.Selector exposing (tag, text)


snapshot : Snapshot
snapshot =
    { selection = "ref: container"
    , groups =
        [ { name = "container", contents = { refs = [ "singleton" ], items = [] } }
        , { name = "singleton", contents = { refs = [], items = [ "Item" ] } }
        ]
    }


current : Main.Model
current =
    let
        base =
            Main.defaultModel
    in
    { base
        | selection = Selection.init (Just snapshot) snapshot.selection Nothing |> Tuple.first
        , reviewTab = Main.Current
        , done = Set.singleton "Item"
    }


step : Main.Msg -> Main.Model -> Main.Model
step msg =
    Main.update msg >> Tuple.first


tests : Test
tests =
    describe "Group items navigation"
        [ test "opening and closing a group preserves list preferences and persisted state" <|
            \_ ->
                let
                    model =
                        { current | viewMode = Main.Done, doneDisplay = Main.Hide, showGroupLinks = True, showContainerGroups = True }

                    opened =
                        step (Main.OpenGroup "container") model
                in
                Expect.equal
                    ( model, Main.serialiseStateForStorage model )
                    ( step Main.BackToList opened, Main.serialiseStateForStorage opened )
        , test "flat lists use raw groups even when container chains collapse out of the overview" <|
            \_ ->
                current
                    |> step (Main.OpenGroup "container")
                    |> Main.resultsView
                    |> Query.fromHtml
                    |> Query.find [ tag "li" ]
                    |> Query.has [ text "Item" ]
        , test "changing review tabs and editing the accepted selection clear group navigation" <|
            \_ ->
                let
                    opened =
                        step (Main.OpenGroup "container") current
                in
                [ step (Main.SelectReviewTab Main.Proposed) opened
                , step (Main.SelectionMsg (Selection.Edit "New item")) opened
                ]
                    |> List.map .selectedGroup
                    |> Expect.equal [ Nothing, Nothing ]
        , test "Apply and Discard clear group navigation even within the Current tab" <|
            \_ ->
                let
                    editedSelection =
                        { selection = "New item"
                        , groups = [ { name = "Individual Items", contents = { refs = [], items = [ "New item" ] } } ]
                        }

                    selection =
                        current.selection

                    readySelection =
                        { selection
                            | draftText = editedSelection.selection
                            , preview = Selection.Ready { revision = selection.revision, request = selection.nextRequest } editedSelection
                        }

                    opened =
                        { current | selection = readySelection, selectedGroup = Just "container" }
                in
                [ step (Main.SelectionMsg Selection.Apply) opened
                , step (Main.SelectionMsg Selection.Discard) opened
                ]
                    |> List.map (\model -> ( model.selectedGroup, model.done ))
                    |> Expect.equal [ ( Nothing, current.done ), ( Nothing, current.done ) ]
        ]
