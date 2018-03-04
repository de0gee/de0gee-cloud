import React from 'react';
import Websocket from 'react-websocket';
import {
  LineChart
} from 'react-easy-chart';

class DataViewer extends React.Component {

  constructor(props) {
    super(props);
    const initialWidth = window.innerWidth > 0 ? window.innerWidth : 500;
    console.log(initialWidth);
    this.state = {
      websocket_url: window.de0gee.websocket_url,
      accelerometer_x: [{x:0,y:0}],
      accelerometer_y: [{x:0,y:0}],
      accelerometer_z: [{x:0,y:0}],
      // gyroscope_x: [{x:0,y:0}],
      // gyroscope_y: [{x:0,y:0}],
      // gyroscope_z: [{x:0,y:0}],
      // magnetometer_x: [{x:0,y:0}],
      // magnetometer_y: [{x:0,y:0}],
      // magnetometer_z: [{x:0,y:0}],
      // temperature: [{x:0,y:0}],
      // ambient_light: [{x:0,y:0}],
      // pressure: [{x:0,y:0}],
      // humidity: [{x:0,y:0}],
      // battery: [{x:0,y:0}],
      showToolTip: false, 
      componentWidth: initialWidth - 100,
    };
  }

  componentDidMount() {
    window.addEventListener('resize', this.handleResize.bind(this));
  }

  componentWillUnmount() {
    window.removeEventListener('resize', this.handleResize);
  }

  handleResize() {
    this.setState({componentWidth: window.innerWidth - 100});
  }

 
  handleData(payload) {
    let result = null;
    try {
      result = JSON.parse(payload);      
    } catch (error1) {
      try {
        result = payload;
      } catch (error) {
        console.log(error1);
        console.log(error);
        console.log(payload);                
      }
      return;
    }
    var currentState = this.state
    if (result.name in currentState) {
      
    } else {
      return
    }

    let values = currentState[result.name]
    if (values.length > 60) {
      values.shift();
    }
    const largestX = values[values.length - 1].x
    values.push({
      x: largestX + 1,
      y: result.data
    })
    
    currentState[result.name] = values
    // this.state[result.name] = values;
    this.setState(currentState)
  }

  render() {
    return ( 
    <div  className="dataviewer">
          <Websocket url = {this.state.websocket_url} onMessage = {this.handleData.bind(this)}/> 
      <h2>Real-time data:</h2>
      <p> Accelerometer </p> 
      <LineChart data = {[this.state.accelerometer_x,this.state.accelerometer_y,this.state.accelerometer_z]}
      width = {this.state.componentWidth}
      height = {this.state.componentWidth / 2}
      interpolate = {'cardinal'}
      axes grid style = {{'.line0': {stroke: 'green'}}}
      /> 
    </div>
    );
  }
}

export default DataViewer;