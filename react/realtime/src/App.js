import React, { Component } from 'react';
import logo from './de0gee-dog.png';
import './App.css';
import ChooseActivity from './ChooseActivity.js'
import DataChooser from './DataChooser.js'
// import ShowUserData from './ShowUserData.js'

class App extends Component {
  render() {
    return (
      <div className="App">
        <header className="App-header">
          <img src={logo} className="App-logo" alt="logo" />
          <h1 className="App-title">de0gee</h1>
        </header>
        {/* <ShowUserData /> */}
        <ChooseActivity />
        <DataChooser />
        
      </div>
    );
  }
}

export default App;
