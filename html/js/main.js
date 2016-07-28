var FileView = React.createClass({

  render: function () {
    return (
      <div className="fileView">
        <h2>FileView for '{this.props.file}'</h2>

        <pre>{this.props.content}</pre>
      </div>
    );
  },

  componentWillMount: function () {
    console.log('componentWillMount');
    var ws = new WebSocket('ws://localhost:8000/ws');

    ws.onerror = function (err) {
      console.log('WebSocket error:');
      console.error(err);
    };

    ws.onopen = function () {
      ws.send('ping');
    };

    ws.onmessage = function (e) {
      console.log('Got message: ' + e.data);
    };

    this.ws = ws;
  },

  componentDidMount: function () {
    console.log('componentDidMount');

  },

});


ReactDOM.render(<FileView file="test" />, document.getElementById('content'));
