module StoredState exposing (StoredState, decoder, empty, encode)

import Dict
import Json.Decode as Decode
import Json.Encode as Encode
import Packing exposing (Snapshot)
import Selection


type alias StoredState =
    { committed : Maybe Snapshot
    , draftText : String
    , legacySelection : Maybe String
    , done : List String
    , doneDisplay : Maybe String
    }


empty : String -> StoredState
empty selection =
    { committed = Nothing
    , draftText = selection
    , legacySelection = Nothing
    , done = []
    , doneDisplay = Nothing
    }


{-| Missing fields may default, but malformed fields must not silently erase data.
-}
optional : String -> a -> Decode.Decoder a -> Decode.Decoder a
optional name fallback valueDecoder =
    Decode.dict Decode.value
        |> Decode.andThen
            (\fields ->
                if Dict.member name fields then
                    Decode.field name valueDecoder

                else
                    Decode.succeed fallback
            )


decoder : String -> Decode.Decoder StoredState
decoder defaultSelection =
    optional "version" 1 Decode.int
        |> Decode.andThen
            (\version ->
                case version of
                    1 ->
                        Decode.map3
                            (\selection done doneDisplay ->
                                { committed = Nothing
                                , draftText = Maybe.withDefault defaultSelection selection
                                , legacySelection = selection
                                , done = done
                                , doneDisplay = doneDisplay
                                }
                            )
                            (optional "selection" Nothing (Decode.nullable Decode.string))
                            (optional "done" [] (Packing.nullableList Decode.string))
                            (optional "doneDisplay" Nothing (Decode.nullable Decode.string))

                    2 ->
                        Decode.map5 StoredState
                            (Decode.field "committed" (Decode.nullable Packing.decodeSnapshot))
                            (Decode.field "draftText" Decode.string)
                            (optional "legacySelection" Nothing (Decode.nullable Decode.string))
                            (optional "done" [] (Packing.nullableList Decode.string))
                            (optional "doneDisplay" Nothing (Decode.nullable Decode.string))

                    _ ->
                        Decode.fail "This saved selection uses an unsupported storage version."
            )


encode : Selection.Model -> List String -> String -> String
encode selection done doneDisplay =
    Encode.object
        [ ( "version", Encode.int 2 )
        , ( "committed", selection.committed |> Maybe.map Packing.encodeSnapshot |> Maybe.withDefault Encode.null )
        , ( "draftText", Encode.string selection.draftText )

        -- Preserve the old accepted text if the page reloads during migration.
        , ( "legacySelection", selection.bootstrap |> Maybe.map (.selection >> Encode.string) |> Maybe.withDefault Encode.null )
        , ( "done", Encode.list Encode.string done )
        , ( "doneDisplay", Encode.string doneDisplay )
        ]
        |> Encode.encode 0
