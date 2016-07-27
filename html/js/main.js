var FileView = React.createClass({

  render: function () {
    return (
      <div className="fileView">
        <h2>FileView</h2>

      </div>
    );
  },

});


ReactDOM.render(<FileView file="test" />, document.getElementById('content'));
