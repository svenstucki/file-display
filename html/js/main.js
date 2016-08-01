var FileView = React.createClass({

  getInitialState: function () {
    return {
      content: ''
    };
  },

  render: function () {
    return (
      <div className="fileView">
        <h2>FileView for '{this.props.file}'</h2>

        <pre>{this.state.content}</pre>
      </div>
    );
  },

  componentWillMount: function () {
    var that = this;

    var ws = new WebSocket('ws://localhost:8000/ws');
    ws.binaryType = 'arraybuffer';

    ws.onerror = function (err) {
      console.log('WebSocket error:');
      console.error(err);
    };

    ws.onopen = function () {
      var cmd = { file: that.props.file };
      ws.send(JSON.stringify(cmd));
    };

    ws.onmessage = function (e) {
      var update = JSON.parse(e.data);

      console.log('Got update:');
      console.log(update);

      if (update.file != that.props.file) {
        console.log('Discarded');
        return;
      }

      that.setState({ content: update.content });
    };

    this.ws = ws;
  },

  componentDidMount: function () {
  },

});


ReactDOM.render(<FileView file="test" />, document.getElementById('content'));
