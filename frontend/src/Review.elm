module Review exposing (view)

import Html exposing (Html, div, h3, h4, li, p, text, ul)
import Html.Attributes exposing (style)
import Packing
import Selection
import SelectionDiff


view : Selection.Model -> Html msg
view model =
    let
        baseline =
            model.committed |> Maybe.withDefault { selection = "", groups = [] }

        itemChanges =
            case Selection.ready model of
                Nothing ->
                    [ p [] [ text "Resolve the current draft to see its item changes. Your current list is unchanged." ] ]

                Just proposed ->
                    let
                        diff =
                            SelectionDiff.compare baseline.groups proposed.groups

                        count xs =
                            String.fromInt (List.length xs)

                        itemList label marker color items =
                            if List.isEmpty items then
                                text ""

                            else
                                div []
                                    [ h4 [] [ text (label ++ " (" ++ count items ++ ")") ]
                                    , ul []
                                        (items
                                            |> List.map
                                                (\item ->
                                                    li [ style "color" color ]
                                                        [ text (marker ++ " " ++ item.name)
                                                        , div [ style "font-size" "0.85em", style "color" "#555" ]
                                                            [ text ("From: " ++ String.join ", " item.sources) ]
                                                        ]
                                                )
                                        )
                                    ]
                    in
                    [ p [ Html.Attributes.attribute "role" "status" ]
                        [ text
                            (count diff.added
                                ++ (if List.length diff.added == 1 then
                                        " item added · "

                                    else
                                        " items added · "
                                   )
                                ++ count diff.removed
                                ++ " removed · "
                                ++ String.fromInt diff.retained
                                ++ " retained"
                            )
                        ]
                    , if List.isEmpty diff.added && List.isEmpty diff.removed then
                        p []
                            [ text
                                (if Selection.hasChanges model then
                                    "Selection changed; no change to overall items."

                                 else
                                    "No unapplied changes."
                                )
                            ]

                      else
                        text ""
                    , if proposed.selection == baseline.selection && Packing.canonical proposed.groups /= Packing.canonical baseline.groups then
                        p [] [ text "Referenced groups have changed since this selection was applied. Review these changes before accepting the updated list." ]

                      else
                        text ""
                    , itemList "Added" "+" "#17603a" diff.added
                    , itemList "Removed" "−" "#a32121" diff.removed
                    , if List.isEmpty diff.sourceChanges then
                        text ""

                      else
                        Html.details []
                            [ Html.summary [] [ text ("Source changes (" ++ count diff.sourceChanges ++ ") — items still included") ]
                            , ul []
                                (diff.sourceChanges
                                    |> List.map
                                        (\item ->
                                            li []
                                                [ text item.name
                                                , div [] [ text ("Before: " ++ String.join ", " item.before) ]
                                                , div [] [ text ("After: " ++ String.join ", " item.after) ]
                                                ]
                                        )
                                )
                            ]
                    ]
    in
    div []
        ([ h3 [] [ text "Changes to resolved items" ] ]
            ++ itemChanges
            ++ [ Html.details []
                    [ Html.summary [] [ text "Selection edits" ]
                    , if baseline.selection == model.draftText then
                        p [] [ text "Selection text is unchanged." ]

                      else
                        Html.pre [ style "white-space" "pre-wrap", style "overflow-wrap" "anywhere" ]
                            (SelectionDiff.lines baseline.selection model.draftText |> List.map lineView)
                    ]
               ]
        )


lineView : SelectionDiff.Line -> Html msg
lineView line =
    let
        ( prefix, content, background ) =
            case line of
                SelectionDiff.Added value ->
                    ( "+ ", value, "#e6f5eb" )

                SelectionDiff.Removed value ->
                    ( "− ", value, "#fce8e8" )

                SelectionDiff.Unchanged value ->
                    ( "  ", value, "transparent" )
    in
    div [ style "background" background ] [ text (prefix ++ content) ]
