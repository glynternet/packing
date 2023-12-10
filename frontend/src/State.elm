port module State exposing (..)

-- In your index.html
--const storedState = localStorage.getItem("appState");
--const app = Elm.Main.init({
--    flags: JSON.parse(storedState)
--});
--app.ports.storeState.subscribe(state => {
--    localStorage.setItem("appState", state);
--    console.log("Stored state");
--});
--TODO(glynernet): can I use Json.Encode.Value here instead?


port storeState : String -> Cmd msg


updateModel : (model -> String) -> model -> ( model, Cmd msg )
updateModel serialise model =
    let
        localStoredState =
            serialise model
    in
    ( model, storeState localStoredState )
