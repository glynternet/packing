module Selection exposing (Effect(..), Model, Msg(..), Note, Preview(..), SimplifyResponse, Token, canApply, hasChanges, init, ready, update)

import Packing exposing (Group, Snapshot)


type alias Token =
    { revision : Int, request : Int }


type Preview
    = Waiting
    | Loading Token
    | Ready Token Snapshot
    | Failed String


type alias Note =
    { refs : List String, items : List String }


type alias SimplifyResponse =
    { selection : String, removedRefs : List String, removedItems : List String }


type alias Model =
    { committed : Maybe Snapshot
    , draftText : String
    , revision : Int
    , nextRequest : Int
    , preview : Preview
    , pendingSimplify : Maybe Token
    , simplifyError : Maybe String
    , simplifyNote : Maybe Note
    , bootstrap : Maybe { token : Token, selection : String }
    , baselineError : Maybe String
    }


type Msg
    = Edit String
    | DebounceElapsed Int
    | Resolve
    | Resolved Token (Result String (List Group))
    | BootstrapResolved Token (Result String (List Group))
    | Rebase Snapshot
    | Apply
    | Discard
    | Simplify
    | Simplified Token (Result String SimplifyResponse)


type Effect
    = Debounce Int
    | Fetch Token String
    | BootstrapFetch Token String
    | SimplifyFetch Token String
    | Persist


init : Maybe Snapshot -> String -> Maybe String -> ( Model, List Effect )
init committed draftText legacySelection =
    let
        model =
            { committed = committed
            , draftText = draftText
            , revision = 0
            , nextRequest = 0
            , preview = Waiting
            , pendingSimplify = Nothing
            , simplifyError = Nothing
            , simplifyNote = Nothing
            , bootstrap = Nothing
            , baselineError = Nothing
            }
    in
    case ( committed, legacySelection ) of
        ( Nothing, Just selection ) ->
            let
                token =
                    { revision = 0, request = 1 }

                base =
                    { model | nextRequest = 1, bootstrap = Just { token = token, selection = selection } }
            in
            if draftText == selection then
                ( base, [ BootstrapFetch token selection ] )

            else
                let
                    ( resolving, effects ) =
                        update Resolve base
                in
                ( resolving, BootstrapFetch token selection :: effects )

        _ ->
            update Resolve model


ready : Model -> Maybe Snapshot
ready model =
    case model.preview of
        Ready token snapshot ->
            if token.revision == model.revision && snapshot.selection == model.draftText then
                Just snapshot

            else
                Nothing

        _ ->
            Nothing


hasChanges : Model -> Bool
hasChanges model =
    case model.committed of
        Nothing ->
            True

        Just baseline ->
            model.draftText
                /= baseline.selection
                || (ready model
                        |> Maybe.map (\snapshot -> Packing.canonical snapshot.groups /= Packing.canonical baseline.groups)
                        |> Maybe.withDefault False
                   )


canApply : Model -> Bool
canApply model =
    ready model
        /= Nothing
        && hasChanges model
        && model.pendingSimplify
        == Nothing
        && model.bootstrap
        == Nothing


update : Msg -> Model -> ( Model, List Effect )
update msg model =
    case msg of
        Edit selection ->
            let
                revision =
                    model.revision + 1
            in
            ( { model
                | draftText = selection
                , revision = revision
                , preview = Waiting
                , pendingSimplify = Nothing
                , simplifyError = Nothing
                , simplifyNote = Nothing
              }
            , [ Debounce revision, Persist ]
            )

        DebounceElapsed revision ->
            if revision == model.revision && model.preview == Waiting then
                update Resolve model

            else
                ( model, [] )

        Resolve ->
            let
                token =
                    { revision = model.revision, request = model.nextRequest + 1 }
            in
            ( { model | preview = Loading token, nextRequest = token.request }
            , [ Fetch token model.draftText ]
            )

        Resolved token result ->
            if model.preview /= Loading token || token.revision /= model.revision then
                -- A newer edit, request, Apply or Discard superseded this response.
                ( model, [] )

            else
                ( { model
                    | preview =
                        case result of
                            Ok groups ->
                                Ready token { selection = model.draftText, groups = groups }

                            Err error ->
                                Failed error
                  }
                , []
                )

        BootstrapResolved token result ->
            case model.bootstrap of
                Just bootstrap ->
                    if bootstrap.token /= token then
                        ( model, [] )

                    else
                        let
                            untouched =
                                model.revision == token.revision && model.draftText == bootstrap.selection && model.preview == Waiting
                        in
                        case result of
                            Ok groups ->
                                let
                                    snapshot =
                                        { selection = bootstrap.selection, groups = groups }
                                in
                                ( { model
                                    | committed = Just snapshot
                                    , bootstrap = Nothing
                                    , baselineError = Nothing
                                    , preview =
                                        if untouched then
                                            Ready token snapshot

                                        else
                                            model.preview
                                  }
                                , [ Persist ]
                                )

                            Err error ->
                                ( { model
                                    | bootstrap = Nothing
                                    , baselineError = Just error
                                    , preview =
                                        if untouched then
                                            Failed error

                                        else
                                            model.preview
                                  }
                                , [ Persist ]
                                )

                Nothing ->
                    ( model, [] )

        Rebase baseline ->
            let
                revision =
                    model.revision + 1

                rebased =
                    { model
                        | committed = Just baseline
                        , revision = revision
                        , preview =
                            case ready model of
                                Just snapshot ->
                                    Ready { revision = revision, request = model.nextRequest } snapshot

                                Nothing ->
                                    Waiting
                        , bootstrap = Nothing
                        , baselineError = Nothing
                        , pendingSimplify = Nothing
                        , simplifyError = Nothing
                        , simplifyNote = Nothing
                    }
            in
            if ready rebased /= Nothing then
                ( rebased, [ Persist ] )

            else
                let
                    ( resolving, effects ) =
                        update Resolve rebased
                in
                ( resolving, Persist :: effects )

        Apply ->
            case ready model of
                Just snapshot ->
                    if canApply model then
                        let
                            revision =
                                model.revision + 1
                        in
                        ( { model
                            | committed = Just snapshot
                            , revision = revision
                            , preview = Ready { revision = revision, request = model.nextRequest } snapshot
                            , pendingSimplify = Nothing
                            , simplifyNote = Nothing
                            , simplifyError = Nothing
                            , baselineError = Nothing
                          }
                        , [ Persist ]
                        )

                    else
                        ( model, [] )

                Nothing ->
                    ( model, [] )

        Discard ->
            case model.committed of
                Just snapshot ->
                    let
                        revision =
                            model.revision + 1
                    in
                    ( { model
                        | draftText = snapshot.selection
                        , revision = revision
                        , preview = Ready { revision = revision, request = model.nextRequest } snapshot
                        , pendingSimplify = Nothing
                        , simplifyNote = Nothing
                        , simplifyError = Nothing
                        , baselineError = Nothing
                        , bootstrap = Nothing
                      }
                    , [ Persist ]
                    )

                Nothing ->
                    ( model, [] )

        Simplify ->
            if model.pendingSimplify /= Nothing then
                ( model, [] )

            else
                let
                    token =
                        { revision = model.revision, request = model.nextRequest + 1 }
                in
                ( { model | pendingSimplify = Just token, nextRequest = token.request, simplifyError = Nothing, simplifyNote = Nothing }
                , [ SimplifyFetch token model.draftText ]
                )

        Simplified token result ->
            if model.pendingSimplify /= Just token || token.revision /= model.revision then
                -- Simplification only transforms the exact draft it was requested for.
                ( model, [] )

            else
                case result of
                    Err error ->
                        ( { model | pendingSimplify = Nothing, simplifyError = Just error }, [] )

                    Ok response ->
                        let
                            ( edited, effects ) =
                                if response.selection == model.draftText then
                                    ( { model | pendingSimplify = Nothing }, [] )

                                else
                                    update (Edit response.selection) model
                        in
                        ( { edited | simplifyNote = Just { refs = response.removedRefs, items = response.removedItems } }, effects )
